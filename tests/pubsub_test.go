package storage_test

import (
	"os"
	"tealis/internal/storage"
	"testing"
	"time"
)

func TestPubSub(t *testing.T) {
	// Setup
	aofFilePath := "./snapshot"
	snapshotPath := "./snapshot"
	// Clean up the ./snapshot folder and files before starting the test

	defer os.Remove(aofFilePath) // Clean up the test AOF file

	// Initialize a Tealis instance with AOF enabled
	r := storage.NewTealis(aofFilePath, snapshotPath, false)

	channel := "test-channel"

	// Simulate a client connection for testing
	clientID := "client1"
	r.AddMockClientConnection(clientID) // Mock connection for the test

	// Test Subscribe
	subscribeResp := r.Subscribe(clientID, channel)
	if subscribeResp != "+SUBSCRIBED to test-channel" {
		t.Errorf("Subscribe failed: expected '+SUBSCRIBED to test-channel', got '%s'", subscribeResp)
	}

	// Test Publish
	message := "Hello, Pub/Sub!"
	publishResp := r.Publish(channel, message)
	if publishResp != ":1" {
		t.Errorf("Publish failed: expected ':1', got '%s'", publishResp)
	}

	// Test Message Delivery
	conn := r.GetMockClientConnection(clientID)
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
	unsubscribeResp := r.Unsubscribe(clientID, channel)
	if unsubscribeResp != "+UNSUBSCRIBED from test-channel" {
		t.Errorf("Unsubscribe failed: expected '+UNSUBSCRIBED from test-channel', got '%s'", unsubscribeResp)
	}

	// Test Publish after Unsubscribe
	publishResp = r.Publish(channel, message)
	if publishResp != ":0" {
		t.Errorf("Publish after unsubscribe failed: expected ':0', got '%s'", publishResp)
	}
}

func TestMultipleSubscribers(t *testing.T) {
	aofFilePath := "test.aof"
	defer os.Remove(aofFilePath) // Clean up the test AOF file

	// Initialize a Tealis instance with AOF enabled
	store := storage.NewTealis(aofFilePath, "./snapshot", false)
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
