package storage_test

import (
	"os"
	"tealis/internal/storage"
	"testing"
)

func TestRedisCloneStreams(t *testing.T) {
	aofFilePath := "test.aof"
	defer os.Remove(aofFilePath) // Clean up the test AOF file

	// Initialize a RedisClone instance with AOF enabled
	store := storage.NewRedisClone(aofFilePath, "", true)
	var id string
	t.Run("XADD - Add entry to stream", func(t *testing.T) {
		id = store.XAdd("mystream", "*", map[string]string{"field1": "value1", "field2": "value2"})
		if id == "" {
			t.Errorf("expected a generated ID, got an empty string")
		}
	})

	t.Run("XLEN - Check stream length", func(t *testing.T) {
		length := store.XLen("mystream")
		if length != 1 {
			t.Errorf("expected stream length 1, got %d", length)
		}
	})

	t.Run("XRANGE - Retrieve entries in range", func(t *testing.T) {
		entries := store.XRange("mystream", "0", "999999999999999")
		if len(entries) != 1 {
			t.Errorf("expected 1 entry, got %d", len(entries))
		}
		if entries[0].Fields["field1"] != "value1" {
			t.Errorf("expected field1 to be 'value1', got '%s'", entries[0].Fields["field1"])
		}
	})

	t.Run("XGROUP CREATE - Create a consumer group", func(t *testing.T) {
		success := store.XGroupCreate("mystream", "mygroup")
		if !success {
			t.Errorf("expected XGROUP CREATE to succeed, got failure")
		}
	})

	t.Run("XREADGROUP - Read entries for a consumer group", func(t *testing.T) {
		entries := store.XReadGroup("mystream", "mygroup", "consumer1", "0", 10)
		if len(entries) != 1 {
			t.Errorf("expected 1 entry, got %d", len(entries))
		}
		if entries[0].Fields["field2"] != "value2" {
			t.Errorf("expected field2 to be 'value2', got '%s'", entries[0].Fields["field2"])
		}
	})

	t.Run("XACK - Acknowledge processed entries", func(t *testing.T) {
		ackCount := store.XAck("mystream", "mygroup", []string{id})
		if ackCount != 1 {
			t.Errorf("expected 1 acknowledged entry, got %d", ackCount)
		}
	})
}
