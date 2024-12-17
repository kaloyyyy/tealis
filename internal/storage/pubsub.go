package storage

import (
	"fmt"
	"log"
)

// Subscribe adds a client to a channel's subscriber list.
func (r *RedisClone) Subscribe(clientID, channel string) string {
	r.mu.Lock()
	defer r.mu.Unlock()

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
	r.mu.Lock()
	defer r.mu.Unlock()

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
	r.mu.RLock()
	defer r.mu.RUnlock()

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
func (r *RedisClone) deliverMessages(clientID string, msgChan chan string) {
	for msg := range msgChan {
		// Check if the client is a real connection
		if conn, ok := r.ClientConnections[clientID]; ok {
			_, err := conn.Write([]byte(msg + "\r\n"))
			if err != nil {
				log.Printf("Error delivering message to %s: %v", clientID, err)
			}
		} else if mockClient, ok := r.mockClients[clientID]; ok {
			// For mock clients, write to the Outbox
			select {
			case mockClient.Outbox <- msg:
			default:
				log.Printf("Mock client %s Outbox is full, dropping message", clientID)
			}
		}
	}
}

func NewMockClientConnection() *MockClientConnection {
	return &MockClientConnection{Outbox: make(chan string, 100)}
}

func (r *RedisClone) AddMockClientConnection(clientID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.mockClients[clientID] = NewMockClientConnection()
}

func (r *RedisClone) GetMockClientConnection(clientID string) *MockClientConnection {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.mockClients[clientID]
}
