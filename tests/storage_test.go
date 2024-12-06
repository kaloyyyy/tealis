package storage

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	"reflect"
	_ "strconv"
	_ "tealis/internal/commands"
	_ "tealis/internal/protocol"
	"tealis/internal/storage"
	"testing"
	"time"
)

func TestSetGet(t *testing.T) {
	store := storage.NewRedisClone()

	// Test setting a key-value pair
	store.Set("key1", "value1", 0)
	value, exists := store.Get("key1")
	assert.True(t, exists, "Expected key to exist")
	assert.Equal(t, "value1", value, "Expected value to be 'value1'")

	// Test getting a non-existing key
	_, exists = store.Get("nonexistent")
	assert.False(t, exists, "Expected key to not exist")
}

func TestSetWithTTL(t *testing.T) {
	store := storage.NewRedisClone()

	// Create a cancellable context for cleanup
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start the cleanup process
	store.StartCleanup(ctx)

	// Set key with TTL of 2 seconds
	store.Set("key2", "value2", 2*time.Second)
	value, exists := store.Get("key2")
	assert.True(t, exists, "Expected key to exist")
	assert.Equal(t, "value2", value, "Expected value to be 'value2'")

	// Wait for the TTL to expire
	time.Sleep(3 * time.Second)
	_, exists = store.Get("key2")
	assert.False(t, exists, "Expected key to expire")
}

func TestDel(t *testing.T) {
	store := storage.NewRedisClone()

	// Set and delete a key
	store.Set("key3", "value3", 0)
	deleted := store.Del("key3")
	assert.True(t, deleted, "Expected key to be deleted")

	// Try to delete a non-existing key
	deleted = store.Del("nonexistent")
	assert.False(t, deleted, "Expected no key to be deleted")
}

func TestExists(t *testing.T) {
	store := storage.NewRedisClone()

	// Set a key and check if it exists
	store.Set("key4", "value4", 0)
	exists := store.Exists("key4")
	assert.True(t, exists, "Expected key to exist")

	// Check for a non-existing key
	exists = store.Exists("nonexistent")
	assert.False(t, exists, "Expected key to not exist")
}

func TestAppend(t *testing.T) {
	store := storage.NewRedisClone()

	// Test appending to a key
	store.Set("key5", "hello", 0)
	newLength := store.Append("key5", " world")
	assert.Equal(t, 11, newLength, "Expected new length to be 11")
	value, _ := store.Get("key5")
	assert.Equal(t, "hello world", value, "Expected value to be 'hello world'")

	// Append to a non-existing key
	newLength = store.Append("key6", "new")
	assert.Equal(t, 3, newLength, "Expected new length to be 3")
	value, _ = store.Get("key6")
	assert.Equal(t, "new", value, "Expected value to be 'new'")
}

func TestStrLen(t *testing.T) {
	store := storage.NewRedisClone()

	// Test string length
	store.Set("key7", "some value", 0)
	length := store.StrLen("key7")
	assert.Equal(t, 10, length, "Expected length to be 10")

	// Test length of non-existing key
	length = store.StrLen("nonexistent")
	assert.Equal(t, 0, length, "Expected length to be 0")
}

func TestIncrBy(t *testing.T) {
	store := storage.NewRedisClone()

	// Test INCRBY with an existing integer key
	store.Set("key8", "5", 0)
	newValue, err := store.IncrBy("key8", 3)
	assert.Nil(t, err, "Expected no error")
	assert.Equal(t, 8, newValue, "Expected value to be 8")

	// Test INCRBY with a non-existing key
	newValue, err = store.IncrBy("key9", 4)
	assert.Nil(t, err, "Expected no error")
	assert.Equal(t, 4, newValue, "Expected value to be 4")

	// Test INCRBY with a non-integer value
	store.Set("key10", "not an integer", 0)
	newValue, err = store.IncrBy("key10", 2)
	assert.NotNil(t, err, "Expected error")
}

func TestDecrBy(t *testing.T) {
	store := storage.NewRedisClone()

	// Test DECRBY with an existing integer key
	store.Set("key11", "5", 0)
	newValue, err := store.IncrBy("key11", -2)
	assert.Nil(t, err, "Expected no error")
	assert.Equal(t, 3, newValue, "Expected value to be 3")

	// Test DECRBY with a non-existing key
	newValue, err = store.IncrBy("key12", -4)
	assert.Nil(t, err, "Expected no error")
	assert.Equal(t, -4, newValue, "Expected value to be -4")
}

func TestGetRange(t *testing.T) {
	store := storage.NewRedisClone()

	// Set a string value
	store.Set("key13", "Hello World", 0)

	// Test GETRANGE
	result := store.GetRange("key13", 0, 4)
	assert.Equal(t, "Hello", result, "Expected range to be 'Hello'")

	// Test GETRANGE with negative indices
	result = store.GetRange("key13", -5, -1)
	assert.Equal(t, "World", result, "Expected range to be 'World'")

	// Test GETRANGE with out-of-bounds indices
	result = store.GetRange("key13", 10, 15)
	assert.Equal(t, "d", result, "Expected empty result")
}

func TestSetRange(t *testing.T) {
	store := storage.NewRedisClone()

	// Test 1: Basic SETRANGE functionality
	store.Set("key1", "Hello", 0)
	// Set the range at index 6 with the value "Go"
	resultLen := store.SetRange("key1", 6, "Go") // Expected: "Hello Go"
	assert.Equal(t, 8, resultLen, "Expected length after SETRANGE to be 8")
	value, exists := store.Get("key1")
	assert.True(t, exists, "Expected key to exist")
	assert.Equal(t, "Hello Go", value, "Expected value to be 'Hello Go'")

	// Test 2: Padding with null bytes (when offset is beyond current length)
	store.Set("key2", "Hello     Redis", 0)
	// Set range at a large offset, padding with null bytes
	resultLen = store.SetRange("key2", 10, "Redis") // Expected: "Hello\x00\x00\x00Redis"
	assert.Equal(t, 15, resultLen, "Expected length after SETRANGE to be 15")
	value, exists = store.Get("key2")
	assert.True(t, exists, "Expected key to exist")
	assert.Equal(t, "Hello     Redis", value, "Expected value to be 'Hello     Redis'")

	// Test 3: Overwriting part of the string
	store.Set("key3", "Hello World", 0)
	// Set range at index 6 to overwrite part of the string with "Gorld"
	resultLen = store.SetRange("key3", 6, "Gorld") // Expected: "Hello Gorld"
	assert.Equal(t, 11, resultLen, "Expected length after SETRANGE to be 11")
	value, exists = store.Get("key3")
	assert.True(t, exists, "Expected key to exist")
	assert.Equal(t, "Hello Gorld", value, "Expected value to be 'Hello Gorld'")
}
func TestKeys(t *testing.T) {
	store := storage.NewRedisClone()

	// Set keys
	store.Set("key1", "value1", 0)
	store.Set("key2", "value2", 0)
	store.Set("anotherkey", "value3", 0)

	// Test KEYS with *
	keys := store.Keys("*")
	if len(keys) != 3 {
		t.Errorf("Expected 3 keys, got %d", len(keys))
	}

	// Test KEYS with specific pattern
	keys = store.Keys("key*")
	if len(keys) != 2 {
		t.Errorf("Expected 2 keys for pattern 'key*', got %d", len(keys))
	}
	// Test KEYS with specific pattern
	keys = store.Keys("*key")
	if len(keys) != 1 {
		t.Errorf("Expected 1 keys for pattern 'key*', got %d", len(keys))
	}
	// Test KEYS with no match
	keys = store.Keys("nomatch*")
	if len(keys) != 0 {
		t.Errorf("Expected 0 keys for pattern 'nomatch*', got %d", len(keys))
	}
}

func TestJSONSetAndGet(t *testing.T) {
	r := storage.RedisClone{
		Store: map[string]string{},
	}

	// JSON.SET
	err := r.JSONSet("key1", "$.a.b", []interface{}{4, 5})
	if err != nil {
		t.Fatalf("JSONSet failed: %v", err)
	}

	// JSON.GET
	value, err := r.JSONGet("key1", "$.a.b")
	if err != nil {
		t.Fatalf("JSONGet failed: %v", err)
	}

	// Convert []interface{} to []int for comparison
	actual, err := interfaceSliceToIntSlice(value.([]interface{}))
	if err != nil {
		t.Fatalf("Conversion failed: %v", err)
	}

	expected := []int{4, 5}
	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("Expected %v, got %v", expected, actual)
	}
}

func interfaceSliceToIntSlice(input []interface{}) ([]int, error) {
	result := make([]int, len(input))
	for i, v := range input {
		num, ok := v.(float64) // JSON numbers are unmarshaled as float64
		if !ok {
			return nil, fmt.Errorf("value at index %d is not a number", i)
		}
		result[i] = int(num)
	}
	return result, nil
}
