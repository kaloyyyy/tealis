package storage_test

import (
	"os"
	"tealis/internal/storage"
	"testing"
)

func TestHSET(t *testing.T) {
	// Setup
	aofFilePath := "./snapshot"
	snapshotPath := "./snapshot"
	// Clean up the ./snapshot folder and files before starting the test

	defer os.Remove(aofFilePath) // Clean up the test AOF file

	// Initialize a RedisClone instance with AOF enabled
	r := storage.NewRedisClone(aofFilePath, snapshotPath, false)

	// Test adding a new field
	result := r.HSET("myhash", "field1", "value1")
	if result != 1 {
		t.Errorf("Expected 1, got %d", result)
	}

	// Test updating an existing field
	result = r.HSET("myhash", "field1", "value2")
	if result != 0 {
		t.Errorf("Expected 0, got %d", result)
	}

	// Test adding another new field
	result = r.HSET("myhash", "field2", "value3")
	if result != 1 {
		t.Errorf("Expected 1, got %d", result)
	}
}

func TestHGET(t *testing.T) {
	// Setup
	aofFilePath := "./snapshot"
	snapshotPath := "./snapshot"

	defer os.Remove(aofFilePath) // Clean up the test AOF file

	// Initialize a RedisClone instance with AOF enabled
	r := storage.NewRedisClone(aofFilePath, snapshotPath, false)

	r.HSET("myhash", "field1", "value1")

	// Test retrieving an existing field
	value, exists := r.HGET("myhash", "field1")
	if !exists || value != "value1" {
		t.Errorf("Expected 'value1', got '%v'", value)
	}

	// Test retrieving a non-existing field
	_, exists = r.HGET("myhash", "field2")
	if exists {
		t.Errorf("Expected field to not exist")
	}
}

func TestHMSET(t *testing.T) {
	// Setup
	aofFilePath := "./snapshot"
	snapshotPath := "./snapshot"

	defer os.Remove(aofFilePath) // Clean up the test AOF file

	// Initialize a RedisClone instance with AOF enabled
	r := storage.NewRedisClone(aofFilePath, snapshotPath, false)

	// Set multiple fields
	r.HMSET("myhash", map[string]interface{}{
		"field1": "value1",
		"field2": "value2",
	})

	// Verify the fields were set
	value, exists := r.HGET("myhash", "field1")
	if !exists || value != "value1" {
		t.Errorf("Expected 'value1', got '%v'", value)
	}

	value, exists = r.HGET("myhash", "field2")
	if !exists || value != "value2" {
		t.Errorf("Expected 'value2', got '%v'", value)
	}
}

func TestHGETALL(t *testing.T) {
	// Setup
	aofFilePath := "./snapshot"
	snapshotPath := "./snapshot"
	// Clean up the ./snapshot folder and files before starting the test

	defer os.Remove(aofFilePath) // Clean up the test AOF file

	// Initialize a RedisClone instance with AOF enabled
	r := storage.NewRedisClone(aofFilePath, snapshotPath, false)

	r.HMSET("myhash", map[string]interface{}{
		"field1": "value1",
		"field2": "value2",
	})

	// Retrieve all fields
	allFields := r.HGETALL("myhash")
	expected := map[string]interface{}{
		"field1": "value1",
		"field2": "value2",
	}

	if len(allFields) != len(expected) {
		t.Errorf("Expected %d fields, got %d", len(expected), len(allFields))
	}

	for field, value := range expected {
		if allFields[field] != value {
			t.Errorf("Expected '%v' for '%s', got '%v'", value, field, allFields[field])
		}
	}
}

func TestHDEL(t *testing.T) {

	// Setup
	aofFilePath := "./snapshot"
	snapshotPath := "./snapshot"
	// Clean up the ./snapshot folder and files before starting the test

	defer os.Remove(aofFilePath) // Clean up the test AOF file

	// Initialize a RedisClone instance with AOF enabled
	r := storage.NewRedisClone(aofFilePath, snapshotPath, false)

	r.HSET("myhash", "field1", "value1")
	r.HSET("myhash", "field2", "value2")

	// Delete an existing field
	result := r.HDEL("myhash", "field1")
	if result != 1 {
		t.Errorf("Expected 1, got %d", result)
	}

	// Verify the field was deleted
	_, exists := r.HGET("myhash", "field1")
	if exists {
		t.Errorf("Expected field1 to be deleted")
	}

	// Attempt to delete a non-existing field
	result = r.HDEL("myhash", "field3")
	if result != 0 {
		t.Errorf("Expected 0, got %d", result)
	}
}

func TestHEXISTS(t *testing.T) {
	// Setup
	aofFilePath := "./snapshot"
	snapshotPath := "./snapshot"
	// Clean up the ./snapshot folder and files before starting the test

	defer os.Remove(aofFilePath) // Clean up the test AOF file

	// Initialize a RedisClone instance with AOF enabled
	r := storage.NewRedisClone(aofFilePath, snapshotPath, false)

	r.HSET("myhash", "field1", "value1")

	// Check existence of an existing field
	exists := r.HEXISTS("myhash", "field1")
	if !exists {
		t.Errorf("Expected field1 to exist")
	}

	// Check existence of a non-existing field
	exists = r.HEXISTS("myhash", "field2")
	if exists {
		t.Errorf("Expected field2 to not exist")
	}
}
