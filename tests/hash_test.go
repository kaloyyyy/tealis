package storage_test

import (
	"tealis/internal/storage"
	"testing"
)

func TestHSET(t *testing.T) {
	store := storage.NewRedisClone()

	// Test adding a new field
	result := store.HSET("myhash", "field1", "value1")
	if result != 1 {
		t.Errorf("Expected 1, got %d", result)
	}

	// Test updating an existing field
	result = store.HSET("myhash", "field1", "value2")
	if result != 0 {
		t.Errorf("Expected 0, got %d", result)
	}

	// Test adding another new field
	result = store.HSET("myhash", "field2", "value3")
	if result != 1 {
		t.Errorf("Expected 1, got %d", result)
	}
}

func TestHGET(t *testing.T) {
	store := storage.NewRedisClone()

	store.HSET("myhash", "field1", "value1")

	// Test retrieving an existing field
	value, exists := store.HGET("myhash", "field1")
	if !exists || value != "value1" {
		t.Errorf("Expected 'value1', got '%v'", value)
	}

	// Test retrieving a non-existing field
	_, exists = store.HGET("myhash", "field2")
	if exists {
		t.Errorf("Expected field to not exist")
	}
}

func TestHMSET(t *testing.T) {
	store := storage.NewRedisClone()

	// Set multiple fields
	store.HMSET("myhash", map[string]interface{}{
		"field1": "value1",
		"field2": "value2",
	})

	// Verify the fields were set
	value, exists := store.HGET("myhash", "field1")
	if !exists || value != "value1" {
		t.Errorf("Expected 'value1', got '%v'", value)
	}

	value, exists = store.HGET("myhash", "field2")
	if !exists || value != "value2" {
		t.Errorf("Expected 'value2', got '%v'", value)
	}
}

func TestHGETALL(t *testing.T) {
	store := storage.NewRedisClone()

	store.HMSET("myhash", map[string]interface{}{
		"field1": "value1",
		"field2": "value2",
	})

	// Retrieve all fields
	allFields := store.HGETALL("myhash")
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
	store := storage.NewRedisClone()

	store.HSET("myhash", "field1", "value1")
	store.HSET("myhash", "field2", "value2")

	// Delete an existing field
	result := store.HDEL("myhash", "field1")
	if result != 1 {
		t.Errorf("Expected 1, got %d", result)
	}

	// Verify the field was deleted
	_, exists := store.HGET("myhash", "field1")
	if exists {
		t.Errorf("Expected field1 to be deleted")
	}

	// Attempt to delete a non-existing field
	result = store.HDEL("myhash", "field3")
	if result != 0 {
		t.Errorf("Expected 0, got %d", result)
	}
}

func TestHEXISTS(t *testing.T) {
	store := storage.NewRedisClone()

	store.HSET("myhash", "field1", "value1")

	// Check existence of an existing field
	exists := store.HEXISTS("myhash", "field1")
	if !exists {
		t.Errorf("Expected field1 to exist")
	}

	// Check existence of a non-existing field
	exists = store.HEXISTS("myhash", "field2")
	if exists {
		t.Errorf("Expected field2 to not exist")
	}
}
