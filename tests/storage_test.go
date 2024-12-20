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
	// Setup
	aofFilePath := "./snapshot"
	snapshotPath := "./snapshot"
	// Clean up the ./snapshot folder and files before starting the test

	defer os.Remove(aofFilePath) // Clean up the test AOF file

	// Initialize a Tealis instance with AOF enabled
	r := storage.NewTealis(aofFilePath, snapshotPath, false)

	// Test setting a key-value pair
	r.Set("key1", "value1", 0)
	value, exists := r.Get("key1")
	assert.True(t, exists, "Expected key to exist")
	assert.Equal(t, "value1", value, "Expected value to be 'value1'")

	// Test getting a non-existing key
	_, exists = r.Get("nonexistent")
	assert.False(t, exists, "Expected key to not exist")

}

func TestSetWithTTL(t *testing.T) {
	// Setup
	aofFilePath := "./snapshot"
	snapshotPath := "./snapshot"
	// Clean up the ./snapshot folder and files before starting the test

	defer os.Remove(aofFilePath) // Clean up the test AOF file

	// Initialize a Tealis instance with AOF enabled
	r := storage.NewTealis(aofFilePath, snapshotPath, false)

	// Create a cancellable context for cleanup
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start the cleanup process
	r.StartCleanup(ctx)

	// Set key with TTL of 2 seconds
	r.Set("key2", "value2", 2*time.Second)
	value, exists := r.Get("key2")
	assert.True(t, exists, "Expected key to exist")
	assert.Equal(t, "value2", value, "Expected value to be 'value2'")

	// Wait for the TTL to expire
	time.Sleep(3 * time.Second)
	_, exists = r.Get("key2")
	assert.False(t, exists, "Expected key to expire")
}

func TestDel(t *testing.T) {
	// Setup
	aofFilePath := "./snapshot"
	snapshotPath := "./snapshot"
	// Clean up the ./snapshot folder and files before starting the test

	defer os.Remove(aofFilePath) // Clean up the test AOF file

	// Initialize a Tealis instance with AOF enabled
	r := storage.NewTealis(aofFilePath, snapshotPath, false)

	// Set and delete a key
	r.Set("key3", "value3", 0)
	deleted := r.Del("key3")
	assert.True(t, deleted, "Expected key to be deleted")

	// Try to delete a non-existing key
	deleted = r.Del("nonexistent")
	assert.False(t, deleted, "Expected no key to be deleted")
}

func TestExists(t *testing.T) {
	// Setup
	aofFilePath := "./snapshot"
	snapshotPath := "./snapshot"
	// Clean up the ./snapshot folder and files before starting the test

	defer os.Remove(aofFilePath) // Clean up the test AOF file

	// Initialize a Tealis instance with AOF enabled
	r := storage.NewTealis(aofFilePath, snapshotPath, false)

	// Set a key and check if it exists
	r.Set("key4", "value4", 0)
	exists := r.Exists("key4")
	assert.True(t, exists, "Expected key to exist")

	// Check for a non-existing key
	exists = r.Exists("nonexistent")
	assert.False(t, exists, "Expected key to not exist")
}

func TestAppend(t *testing.T) {
	// Setup
	aofFilePath := "./snapshot"
	snapshotPath := "./snapshot"
	// Clean up the ./snapshot folder and files before starting the test

	defer os.Remove(aofFilePath) // Clean up the test AOF file

	// Initialize a Tealis instance with AOF enabled
	r := storage.NewTealis(aofFilePath, snapshotPath, false)

	// Test appending to a key
	r.Set("key5", "hello", 0)
	newLength := r.Append("key5", " world")
	assert.Equal(t, 11, newLength, "Expected new length to be 11")
	value, _ := r.Get("key5")
	assert.Equal(t, "hello world", value, "Expected value to be 'hello world'")

	// Append to a non-existing key
	newLength = r.Append("key6", "new")
	assert.Equal(t, 3, newLength, "Expected new length to be 3")
	value, _ = r.Get("key6")
	assert.Equal(t, "new", value, "Expected value to be 'new'")
}

func TestStrLen(t *testing.T) {
	// Setup
	aofFilePath := "./snapshot"
	snapshotPath := "./snapshot"
	// Clean up the ./snapshot folder and files before starting the test

	defer os.Remove(aofFilePath) // Clean up the test AOF file

	// Initialize a Tealis instance with AOF enabled
	r := storage.NewTealis(aofFilePath, snapshotPath, false)

	// Test string length
	r.Set("key7", "some value", 0)
	length := r.StrLen("key7")
	assert.Equal(t, 10, length, "Expected length to be 10")

	// Test length of non-existing key
	length = r.StrLen("nonexistent")
	assert.Equal(t, 0, length, "Expected length to be 0")
}

func TestIncrBy(t *testing.T) {
	// Setup
	aofFilePath := "./snapshot"
	snapshotPath := "./snapshot"
	// Clean up the ./snapshot folder and files before starting the test

	defer os.Remove(aofFilePath) // Clean up the test AOF file

	// Initialize a Tealis instance with AOF enabled
	r := storage.NewTealis(aofFilePath, snapshotPath, false)

	// Test INCRBY with an existing integer key
	r.Set("key8", "5", 0)
	newValue, err := r.IncrBy("key8", 3)
	assert.Nil(t, err, "Expected no error")
	assert.Equal(t, 8, newValue, "Expected value to be 8")

	// Test INCRBY with a non-existing key
	newValue, err = r.IncrBy("key9", 4)
	assert.Nil(t, err, "Expected no error")
	assert.Equal(t, 4, newValue, "Expected value to be 4")

	// Test INCRBY with a non-integer value
	r.Set("key10", "not an integer", 0)
	newValue, err = r.IncrBy("key10", 2)
	assert.NotNil(t, err, "Expected error")
}

func TestDecrBy(t *testing.T) {
	// Setup
	aofFilePath := "./snapshot"
	snapshotPath := "./snapshot"
	// Clean up the ./snapshot folder and files before starting the test

	defer os.Remove(aofFilePath) // Clean up the test AOF file

	// Initialize a Tealis instance with AOF enabled
	r := storage.NewTealis(aofFilePath, snapshotPath, false)

	// Test DECRBY with an existing integer key
	r.Set("key11", "5", 0)
	newValue, err := r.IncrBy("key11", -2)
	assert.Nil(t, err, "Expected no error")
	assert.Equal(t, 3, newValue, "Expected value to be 3")

	// Test DECRBY with a non-existing key
	newValue, err = r.IncrBy("key12", -4)
	assert.Nil(t, err, "Expected no error")
	assert.Equal(t, -4, newValue, "Expected value to be -4")
}

func TestGetRange(t *testing.T) {
	// Setup
	aofFilePath := "./snapshot"
	snapshotPath := "./snapshot"
	// Clean up the ./snapshot folder and files before starting the test

	defer os.Remove(aofFilePath) // Clean up the test AOF file

	// Initialize a Tealis instance with AOF enabled
	r := storage.NewTealis(aofFilePath, snapshotPath, false)
	// Set a string value
	r.Set("key13", "Hello World", 0)

	// Test GETRANGE
	result := r.GetRange("key13", 0, 4)
	assert.Equal(t, "Hello", result, "Expected range to be 'Hello'")

	// Test GETRANGE with negative indices
	result = r.GetRange("key13", -5, -1)
	assert.Equal(t, "World", result, "Expected range to be 'World'")

	// Test GETRANGE with out-of-bounds indices
	result = r.GetRange("key13", 10, 15)
	assert.Equal(t, "d", result, "Expected empty result")
}

func TestSetRange(t *testing.T) {
	// Setup
	aofFilePath := "./snapshot"
	snapshotPath := "./snapshot"
	// Clean up the ./snapshot folder and files before starting the test

	defer os.Remove(aofFilePath) // Clean up the test AOF file

	// Initialize a Tealis instance with AOF enabled
	r := storage.NewTealis(aofFilePath, snapshotPath, false)

	// Test 1: Basic SETRANGE functionality
	r.Set("key1", "Hello", 0)
	// Set the range at index 6 with the value "Go"
	resultLen := r.SetRange("key1", 6, "Go") // Expected: "Hello Go"
	assert.Equal(t, 8, resultLen, "Expected length after SETRANGE to be 8")
	value, exists := r.Get("key1")
	assert.True(t, exists, "Expected key to exist")
	assert.Equal(t, "Hello Go", value, "Expected value to be 'Hello Go'")

	// Test 2: Padding with null bytes (when offset is beyond current length)
	r.Set("key2", "Hello     Redis", 0)
	// Set range at a large offset, padding with null bytes
	resultLen = r.SetRange("key2", 10, "Redis") // Expected: "Hello\x00\x00\x00Redis"
	assert.Equal(t, 15, resultLen, "Expected length after SETRANGE to be 16")
	value, exists = r.Get("key2")
	assert.True(t, exists, "Expected key to exist")
	assert.Equal(t, "Hello     Redis", value, "Expected value to be 'Hello     Redis'")

	// Test 3: Overwriting part of the string
	r.Set("key3", "Hello World", 0)
	// Set range at index 6 to overwrite part of the string with "Gorld"
	resultLen = r.SetRange("key3", 6, "Gorld") // Expected: "Hello Gorld"
	assert.Equal(t, 11, resultLen, "Expected length after SETRANGE to be 12")
	value, exists = r.Get("key3")
	assert.True(t, exists, "Expected key to exist")
	assert.Equal(t, "Hello Gorld", value, "Expected value to be 'Hello Gorld'")
}
