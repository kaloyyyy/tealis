package storage

import (
	"os"
	"tealis/internal/storage"
	"testing"
	"time"
)

func TestAOF(t *testing.T) {
	// Setup
	aofFilePath := "test.aof"
	defer os.Remove(aofFilePath) // Clean up the test AOF file

	// Initialize a RedisClone instance with AOF enabled
	r := storage.NewRedisClone(aofFilePath, "./snapshot", true)

	// Perform operations that modify the database
	r.Set("key1", "value1", 0)
	r.Set("key2", "value2", 0)
	r.EX("key1", 100*time.Second)
	r.Del("key2")

	// Ensure the operations are logged in the AOF file
	r.AofFile.Close() // Ensure all data is flushed to the file
	err := r.SaveSnapshot()
	if err != nil {
		t.Errorf("Error saving aof file: %v", err)
	}
	// Create a new RedisClone instance and reload the AOF file
	r2 := storage.NewRedisClone(aofFilePath, "./snapshot", true)
	err = r2.LoadSnapshot()
	if err != nil {
		t.Errorf("Error loading aof file: %v", err)
	}
	// Verify the state of the new instance
	if v, ok := r2.Store["key1"].(string); !ok || v != "value1" {
		t.Errorf("Expected key1 to have value 'value1', got '%v'", v)
	}

	// Verify that key2 does not exist
	if _, exists := r2.Store["key2"]; exists {
		t.Errorf("Expected key2 to be deleted")
	}
	time.Sleep(1 * time.Second)
	// Verify expiry times (optional)
	expiry, exists := r2.Expiries["key1"]
	timeNow := time.Now()
	if !exists {
		t.Errorf("Expected key1 to have an expiry")
	} else if expiry.Before(timeNow) {
		t.Errorf("Expected key1 expiry to be in the future, got %v vs now: %v", expiry, timeNow)
		t.Errorf("Time difference: %v", timeNow.Sub(expiry))
	}
}
