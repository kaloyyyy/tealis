package storage

import (
	"os"
	_ "sync"
	"tealis/internal/storage"
	"testing"
)

func TestSetBitfield(t *testing.T) {
	// Setup
	aofFilePath := "test.aof"
	defer os.Remove(aofFilePath) // Clean up the test AOF file

	// Initialize a RedisClone instance with AOF enabled
	r := storage.NewRedisClone(aofFilePath, "./snapshot", true)

	// Test setting an i8 value
	err := r.SetBitfield("key1", "i8", 0, -128)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Test setting a value out of range for i8
	err = r.SetBitfield("key1", "i8", 0, 200)
	if err == nil {
		t.Errorf("expected error for out-of-range value, got nil")
	}

	// Test setting a u16 value
	err = r.SetBitfield("key2", "u16", 0, 65535)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Test setting a value out of range for u16
	err = r.SetBitfield("key2", "u16", 0, -1)
	if err == nil {
		t.Errorf("expected error for out-of-range value, got nil")
	}
}

func TestGetBitfield(t *testing.T) {
	aofFilePath := "test.aof"
	defer os.Remove(aofFilePath) // Clean up the test AOF file

	// Initialize a RedisClone instance with AOF enabled
	r := storage.NewRedisClone(aofFilePath, "./snapshot", true)
	// Set up initial bitfield values
	err := r.SetBitfield("key1", "i8", 0, -128)
	if err != nil {
		return
	}
	err = r.SetBitfield("key2", "u16", 0, 65535)
	if err != nil {
		return
	}

	// Test getting an i8 value
	value, err := r.GetBitfield("key1", "i8", 0)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if value != -128 {
		t.Errorf("expected value -128, got %d", value)
	}

	// Test getting a u16 value
	value, err = r.GetBitfield("key2", "u16", 0)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if value != 65535 {
		t.Errorf("expected value 65535, got %d", value)
	}
}

func TestIncrByBitfield(t *testing.T) {
	// Setup
	aofFilePath := "./snapshot/aof.txt"
	snapshotPath := "./snapshot/snapshot.txt"
	defer os.Remove(aofFilePath) // Clean up the test AOF file

	// Initialize a RedisClone instance with AOF enabled
	r := storage.NewRedisClone(aofFilePath, snapshotPath, true)
	store := storage.NewRedisClone(aofFilePath, snapshotPath, true)
	print(store)
	print(r)
	// Set up initial bitfield values
	err := r.SetBitfield("key1", "i8", 0, -128)
	if err != nil {
		return
	}
	err = r.SetBitfield("key2", "u16", 0, 65534)
	if err != nil {
		return
	}

	// Increment an i8 value
	newValue, err := r.IncrByBitfield("key1", "i8", 0, 1)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if newValue != -127 {
		t.Errorf("expected value -127, got %d", newValue)
	}

	// Increment a u16 value
	newValue, err = r.IncrByBitfield("key2", "u16", 0, 1)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if newValue != 65535 {
		t.Errorf("expected value 65535, got %d", newValue)
	}

	// Test overflow for u16
	_, err = r.IncrByBitfield("key2", "u16", 0, 1)
	if err == nil {
		t.Errorf("expected error for overflow, got nil")
	}
}
