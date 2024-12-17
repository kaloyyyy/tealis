package storage

import (
	"context"
	"github.com/stretchr/testify/assert"
	"os"
	_ "strconv"

	_ "tealis/internal/protocol"
	"tealis/internal/storage"
	"testing"
	"time"
)

func TestSetGet(t *testing.T) {
	aofFilePath := "test.aof"
	defer os.Remove(aofFilePath) // Clean up the test AOF file

	// Initialize a RedisClone instance with AOF enabled
	store := storage.NewRedisClone(aofFilePath, "", true)

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
	aofFilePath := "test.aof"
	defer os.Remove(aofFilePath) // Clean up the test AOF file

	// Initialize a RedisClone instance with AOF enabled
	store := storage.NewRedisClone(aofFilePath, "", true)

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
	aofFilePath := "test.aof"
	defer os.Remove(aofFilePath) // Clean up the test AOF file

	// Initialize a RedisClone instance with AOF enabled
	store := storage.NewRedisClone(aofFilePath, "", true)

	// Set and delete a key
	store.Set("key3", "value3", 0)
	deleted := store.Del("key3")
	assert.True(t, deleted, "Expected key to be deleted")

	// Try to delete a non-existing key
	deleted = store.Del("nonexistent")
	assert.False(t, deleted, "Expected no key to be deleted")
}

func TestExists(t *testing.T) {
	aofFilePath := "test.aof"
	defer os.Remove(aofFilePath) // Clean up the test AOF file

	// Initialize a RedisClone instance with AOF enabled
	store := storage.NewRedisClone(aofFilePath, "", true)

	// Set a key and check if it exists
	store.Set("key4", "value4", 0)
	exists := store.Exists("key4")
	assert.True(t, exists, "Expected key to exist")

	// Check for a non-existing key
	exists = store.Exists("nonexistent")
	assert.False(t, exists, "Expected key to not exist")
}

func TestAppend(t *testing.T) {
	aofFilePath := "test.aof"
	defer os.Remove(aofFilePath) // Clean up the test AOF file

	// Initialize a RedisClone instance with AOF enabled
	store := storage.NewRedisClone(aofFilePath, "", true)

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
	aofFilePath := "test.aof"
	defer os.Remove(aofFilePath) // Clean up the test AOF file

	// Initialize a RedisClone instance with AOF enabled
	store := storage.NewRedisClone(aofFilePath, "", true)

	// Test string length
	store.Set("key7", "some value", 0)
	length := store.StrLen("key7")
	assert.Equal(t, 10, length, "Expected length to be 10")

	// Test length of non-existing key
	length = store.StrLen("nonexistent")
	assert.Equal(t, 0, length, "Expected length to be 0")
}

func TestIncrBy(t *testing.T) {
	aofFilePath := "test.aof"
	defer os.Remove(aofFilePath) // Clean up the test AOF file

	// Initialize a RedisClone instance with AOF enabled
	store := storage.NewRedisClone(aofFilePath, "", true)

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
	aofFilePath := "test.aof"
	defer os.Remove(aofFilePath) // Clean up the test AOF file

	// Initialize a RedisClone instance with AOF enabled
	store := storage.NewRedisClone(aofFilePath, "", true)
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
	aofFilePath := "test.aof"
	defer os.Remove(aofFilePath) // Clean up the test AOF file

	// Initialize a RedisClone instance with AOF enabled
	store := storage.NewRedisClone(aofFilePath, "", true)
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
	aofFilePath := "test.aof"
	defer os.Remove(aofFilePath) // Clean up the test AOF file

	// Initialize a RedisClone instance with AOF enabled
	store := storage.NewRedisClone(aofFilePath, "", true)
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
	assert.Equal(t, 15, resultLen, "Expected length after SETRANGE to be 16")
	value, exists = store.Get("key2")
	assert.True(t, exists, "Expected key to exist")
	assert.Equal(t, "Hello     Redis", value, "Expected value to be 'Hello     Redis'")

	// Test 3: Overwriting part of the string
	store.Set("key3", "Hello World", 0)
	// Set range at index 6 to overwrite part of the string with "Gorld"
	resultLen = store.SetRange("key3", 6, "Gorld") // Expected: "Hello Gorld"
	assert.Equal(t, 11, resultLen, "Expected length after SETRANGE to be 12")
	value, exists = store.Get("key3")
	assert.True(t, exists, "Expected key to exist")
	assert.Equal(t, "Hello Gorld", value, "Expected value to be 'Hello Gorld'")
}
