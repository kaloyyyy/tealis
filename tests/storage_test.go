package storage

import (
	"awesomeProject/internal/storage"
	"testing"
	"time"
)

func TestSetAndGet(t *testing.T) {
	// Initialize a new Redis clone instance
	store := storage.NewRedisClone()

	// Set a key-value pair with no expiration (0 means no expiration)
	store.Set("key", "value", 0)

	// Retrieve the value associated with the key
	val, exists := store.Get("key")

	// Check if the key exists and the value matches
	if !exists || val != "value" {
		t.Errorf("Expected 'value', got '%s'", val)
	}
}

func TestExpiration(t *testing.T) {
	// Initialize a new Redis clone instance
	store := storage.NewRedisClone()

	// Start cleanup in a separate goroutine
	store.StartCleanup()

	// Set a key-value pair with a 2-second expiration time
	store.Set("key", "value", 2*time.Second)

	// Sleep for 3 seconds to allow the key to expire
	time.Sleep(3 * time.Second)

	// Try to get the key after it should have expired
	_, exists := store.Get("key")
	if exists {
		t.Errorf("Expected key to be expired")
	}
}
