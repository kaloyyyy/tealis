package storage

import (
	"fmt"
	"github.com/gorilla/websocket"
	"log"
	"net"
)

// Subscribe adds a client to a channel's subscriber list.
func (r *RedisClone) Subscribe(clientID, channel string) string {
	r.Mu.Lock()
	defer r.Mu.Unlock()

	if _, exists := r.pubsubSubscribers[channel]; !exists {
		r.pubsubSubscribers[channel] = make(map[string]chan string)
	}

	if _, exists := r.pubsubSubscribers[channel][clientID]; !exists {
		r.pubsubSubscribers[channel][clientID] = make(chan string, 100)        // Buffered channel
		go r.deliverMessages(clientID, r.pubsubSubscribers[channel][clientID]) // Start delivering messages
	}

	return fmt.Sprintf("+SUBSCRIBED to %s", channel)
}

// Unsubscribe removes a client from a channel's subscriber list.
func (r *RedisClone) Unsubscribe(clientID, channel string) string {
	r.Mu.Lock()
	defer r.Mu.Unlock()

	if subs, exists := r.pubsubSubscribers[channel]; exists {
		if msgChan, ok := subs[clientID]; ok {
			close(msgChan)
			delete(subs, clientID)
		}
		if len(subs) == 0 {
			delete(r.pubsubSubscribers, channel) // Remove empty channels
		}
	}
	return fmt.Sprintf("+UNSUBSCRIBED from %s", channel)
}

// Publish sends a message to all subscribers of a channel.
func (r *RedisClone) Publish(channel, message string) string {
	r.Mu.RLock()
	defer r.Mu.RUnlock()

	subscribers, exists := r.pubsubSubscribers[channel]
	if !exists {
		return ":0" // No subscribers
	}

	for _, msgChan := range subscribers {
		select {
		case msgChan <- message:
		default:
			// Drop message if buffer is full to avoid blocking
		}
	}

	return fmt.Sprintf(":%d", len(subscribers))
}

// deliverMessages sends messages from a channel to a client.
// deliverMessages sends messages from a channel to a client.
func (r *RedisClone) deliverMessages(clientID string, msgChan chan string) {
	log.Printf("delivering messages to %s in chan: %v", clientID, msgChan)
	for msg := range msgChan {
		// Check if the client is a WebSocket connection
		if conn, ok := r.ClientConnections[clientID].(*websocket.Conn); ok {
			// Send message to WebSocket client
			err := conn.WriteMessage(websocket.TextMessage, []byte(msg))
			if err != nil {
				log.Printf("Error delivering message to WebSocket client %s: %v", clientID, err)
				// Handle cleanup if needed (e.g., unsubscribe client)
				r.Unsubscribe(clientID, "channel") // Clean up the client subscription
				break
			}
		} else if conn, ok := r.ClientConnections[clientID].(net.Conn); ok {
			// Send message to regular TCP client
			_, err := conn.Write([]byte(msg + "\r\n"))
			if err != nil {
				log.Printf("Error delivering message to TCP client %s: %v", clientID, err)
				// Handle cleanup if needed (e.g., unsubscribe client)
				r.Unsubscribe(clientID, "channel") // Clean up the client subscription
				break
			}
		} else if mockConn, ok := r.mockClients[clientID]; ok {
			// Send message to mock client (i.e., add to Outbox channel)
			select {
			case mockConn.Outbox <- msg:
				// Successfully sent the message to the mock client
			default:
				// If the Outbox is full, handle the case
				log.Printf("Warning: Mock client's Outbox is full, dropping message")
			}
		} else {
			log.Printf("Client %s is neither WebSocket nor TCP, skipping message delivery", clientID)
		}
	}
}

// MockClientConnection simulates a client connection with an Outbox for receiving messages
type MockClientConnection struct {
	Outbox chan string
}

func NewMockClientConnection() *MockClientConnection {
	return &MockClientConnection{Outbox: make(chan string, 100)}
}

func (r *RedisClone) AddMockClientConnection(clientID string) {
	r.Mu.Lock()
	defer r.Mu.Unlock()
	r.mockClients[clientID] = NewMockClientConnection()
}

func (r *RedisClone) GetMockClientConnection(clientID string) *MockClientConnection {
	r.Mu.RLock()
	defer r.Mu.RUnlock()
	return r.mockClients[clientID]
}
