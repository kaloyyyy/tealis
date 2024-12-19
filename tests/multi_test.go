package storage

import (
	"tealis/internal/storage"
	"testing"
)

func TestMultiExecAndDiscard(t *testing.T) {
	// Setup paths
	aofFilePath := "./snapshot"
	snapshotPath := "./snapshot"

	// Initialize a RedisClone instance
	r := storage.NewRedisClone(aofFilePath, snapshotPath, false)

	clientID := "test_client"
	// Begin a transaction
	r.MULTI(clientID)

	// Queue commands within the transaction
	r.Transactions[clientID] = append(r.Transactions[clientID], "SET key1 value1")
	r.Transactions[clientID] = append(r.Transactions[clientID], "SET key2 value2")
	r.Transactions[clientID] = append(r.Transactions[clientID], "DEL key3")

	// Execute the transaction
	response := r.EXEC(clientID)

	// Validate responses from EXEC
	expectedResponse := "+OK\r\n+OK\r\n:0\r\n" // Expected Redis responses
	if response != expectedResponse {
		t.Errorf("EXEC response mismatch. Expected: %q, Got: %q", expectedResponse, response)
	}

	// Ensure commands were applied to the database
	if v, ok := r.Store["key1"].(string); !ok || v != "value1" {
		t.Errorf("Expected key1 to have value 'value1', got '%v'", v)
	}
	if v, ok := r.Store["key2"].(string); !ok || v != "value2" {
		t.Errorf("Expected key2 to have value 'value2', got '%v'", v)
	}
	if _, exists := r.Store["key3"]; exists {
		t.Errorf("Expected key3 to be deleted")
	}

	// Start another transaction
	r.MULTI(clientID)

	// Queue commands
	r.Transactions[clientID] = append(r.Transactions[clientID], "SET key4 value4")
	r.Transactions[clientID] = append(r.Transactions[clientID], "SET key5 value5")

	// Discard the transaction
	r.DISCARD(clientID)

	// Verify that no commands were executed
	if _, exists := r.Store["key4"]; exists {
		t.Errorf("Expected key4 not to be set after DISCARD")
	}
	if _, exists := r.Store["key5"]; exists {
		t.Errorf("Expected key5 not to be set after DISCARD")
	}

	// Ensure no transaction remains for the client
	if _, exists := r.Transactions[clientID]; exists {
		t.Errorf("Expected no active transaction for client %s after DISCARD", clientID)
	}
}
