package storage_test

import (
	"tealis/internal/storage"
	"testing"
	"time"
)

func TestPubSub(t *testing.T) {
	store := storage.NewRedisClone()
	channel := "test-channel"

	// Simulate a client connection for testing
	clientID := "client1"
	store.AddMockClientConnection(clientID) // Mock connection for the test

	// Test Subscribe
	subscribeResp := store.Subscribe(clientID, channel)
	if subscribeResp != "+SUBSCRIBED to test-channel" {
		t.Errorf("Subscribe failed: expected '+SUBSCRIBED to test-channel', got '%s'", subscribeResp)
	}

	// Test Publish
	message := "Hello, Pub/Sub!"
	publishResp := store.Publish(channel, message)
	if publishResp != ":1" {
		t.Errorf("Publish failed: expected ':1', got '%s'", publishResp)
	}

	// Test Message Delivery
	conn := store.GetMockClientConnection(clientID)
	if conn == nil {
		t.Fatal("Failed to retrieve mock client connection")
	}

	select {
	case msg := <-conn.Outbox:
		if msg != message {
			t.Errorf("Expected message '%s', got '%s'", message, msg)
		}
	case <-time.After(time.Second):
		t.Fatal("Message delivery timed out")
	}

	// Test Unsubscribe
	unsubscribeResp := store.Unsubscribe(clientID, channel)
	if unsubscribeResp != "+UNSUBSCRIBED from test-channel" {
		t.Errorf("Unsubscribe failed: expected '+UNSUBSCRIBED from test-channel', got '%s'", unsubscribeResp)
	}

	// Test Publish after Unsubscribe
	publishResp = store.Publish(channel, message)
	if publishResp != ":0" {
		t.Errorf("Publish after unsubscribe failed: expected ':0', got '%s'", publishResp)
	}
}

func TestMultipleSubscribers(t *testing.T) {
	store := storage.NewRedisClone()
	channel := "test-channel"

	// Add multiple mock clients
	clientIDs := []string{"client1", "client2", "client3"}
	for _, clientID := range clientIDs {
		store.AddMockClientConnection(clientID)
		store.Subscribe(clientID, channel)
	}

	// Publish a message
	message := "Hello, all subscribers!"
	publishResp := store.Publish(channel, message)
	if publishResp != ":3" {
		t.Errorf("Publish failed: expected ':3', got '%s'", publishResp)
	}

	// Verify message delivery for each client
	for _, clientID := range clientIDs {
		conn := store.GetMockClientConnection(clientID)
		if conn == nil {
			t.Fatalf("Failed to retrieve mock client connection for %s", clientID)
		}

		select {
		case msg := <-conn.Outbox:
			if msg != message {
				t.Errorf("Expected message '%s', got '%s' for %s", message, msg, clientID)
			}
		case <-time.After(time.Second):
			t.Fatalf("Message delivery timed out for %s", clientID)
		}
	}
}
