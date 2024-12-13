package storage

import (
	"tealis/internal/storage"
	"testing"
)

func TestSETBIT(t *testing.T) {
	r := storage.NewRedisClone()
	key := "key1"

	// Set a bit and verify the previous value
	prev := r.SETBIT(key, 5, 1)
	if prev != 0 {
		t.Errorf("Expected previous bit value to be 0, got %d", prev)
	}

	// Set the same bit to a different value and verify the previous value
	prev = r.SETBIT(key, 5, 0)
	if prev != 1 {
		t.Errorf("Expected previous bit value to be 1, got %d", prev)
	}
}

func TestGETBIT(t *testing.T) {
	r := storage.NewRedisClone()
	key := "key1"

	// Verify the initial state of the bit (should be 0)
	bit := r.GETBIT(key, 10)
	if bit != 0 {
		t.Errorf("Expected bit value to be 0, got %d", bit)
	}

	// Set a bit and verify it can be retrieved correctly
	r.SETBIT(key, 10, 1)
	bit = r.GETBIT(key, 10)
	if bit != 1 {
		t.Errorf("Expected bit value to be 1, got %d", bit)
	}
}

func TestBITCOUNT(t *testing.T) {
	r := storage.NewRedisClone()
	key := "key1"

	// Initially, the count of set bits should be 0
	count := r.BITCOUNT(key)
	if count != 0 {
		t.Errorf("Expected bit count to be 0, got %d", count)
	}

	// Set some bits and verify the count
	r.SETBIT(key, 1, 1)
	r.SETBIT(key, 3, 1)
	r.SETBIT(key, 5, 1)
	count = r.BITCOUNT(key)
	if count != 3 {
		t.Errorf("Expected bit count to be 3, got %d", count)
	}
}

func TestBITOP(t *testing.T) {
	r := storage.NewRedisClone()
	key1 := "key1"
	key2 := "key2"
	destKey := "dest"

	// Set some bits in key1 and key2
	r.SETBIT(key1, 1, 1)
	r.SETBIT(key1, 3, 1)
	r.SETBIT(key2, 3, 1)
	r.SETBIT(key2, 5, 1)

	// Perform AND operation
	r.BITOP("AND", destKey, key1, key2)
	if r.BITCOUNT(destKey) != 1 {
		t.Errorf("Expected BITCOUNT for AND operation to be 1, got %d", r.BITCOUNT(destKey))
	}

	// Perform OR operation
	r.BITOP("OR", destKey, key1, key2)
	if r.BITCOUNT(destKey) != 4 {
		t.Errorf("Expected BITCOUNT for OR operation to be 4, got %d", r.BITCOUNT(destKey))
	}

	// Perform XOR operation
	r.BITOP("XOR", destKey, key1, key2)
	if r.BITCOUNT(destKey) != 3 {
		t.Errorf("Expected BITCOUNT for XOR operation to be 3, got %d", r.BITCOUNT(destKey))
	}

	// Perform NOT operation on key1
	r.BITOP("NOT", destKey, key1)
	// Verify result length matches key1
	result, _ := r.Store[destKey].([]byte)
	if len(result)*8 != 64 { // Assuming default length of 64 bits
		t.Errorf("Expected result length for NOT operation to match key1, got %d bits", len(result)*8)
	}
}
