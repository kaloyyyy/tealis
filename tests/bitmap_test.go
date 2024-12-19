package storage

import (
	"os"
	"tealis/internal/storage"
	"testing"
)

func TestBitFunctions(t *testing.T) {
	// Setup
	aofFilePath := "./snapshot"
	snapshotPath := "./snapshot"

	defer os.Remove(aofFilePath) // Clean up the test AOF file

	// Initialize a RedisClone instance with AOF enabled
	r := storage.NewRedisClone(aofFilePath, snapshotPath, false)

	print(r)
	key1 := "key1"
	key3 := "key3"
	destKey := "dest"

	// Test SETBIT
	prev := r.SETBIT(key1, 5, 1)
	if prev != 0 {
		t.Errorf("SETBIT: Expected previous bit value to be 0, got %d", prev)
	}
	prev = r.SETBIT(key1, 5, 0)
	if prev != 1 {
		t.Errorf("SETBIT: Expected previous bit value to be 1, got %d", prev)
	}

	// Test GETBIT
	bit := r.GETBIT(key1, 10)
	if bit != 0 {
		t.Errorf("GETBIT: Expected bit value to be 0, got %d", bit)
	}
	r.SETBIT(key1, 10, 1)
	bit = r.GETBIT(key1, 10)
	if bit != 1 {
		t.Errorf("GETBIT: Expected bit value to be 1, got %d", bit)
	}

	// Test BITCOUNT
	count := r.BITCOUNT(key1)
	if count != 1 {
		t.Errorf("BITCOUNT: Expected bit count to be 1, got %d", count)
	}
	r.SETBIT(key1, 1, 1)
	r.SETBIT(key1, 3, 1)
	r.SETBIT(key1, 5, 1)
	count = r.BITCOUNT(key1)
	if count != 4 {
		t.Errorf("BITCOUNT: Expected bit count to be 4, got %d", count)
	}
	key4 := "key4"
	// Test BITOP AND
	r.SETBIT(key4, 0, 1)

	r.SETBIT(key4, 2, 1)

	r.SETBIT(key3, 0, 1)
	r.SETBIT(key3, 1, 1)
	r.BITOP("AND", destKey, key3, key4)
	newvalue := 5 & 3
	print(newvalue)
	if r.BITCOUNT(destKey) != 1 {
		t.Errorf("BITOP AND: Expected BITCOUNT to be 1, got %d", r.BITCOUNT(destKey))
	}
	r.SETBIT(key3, 0, 1)
	r.SETBIT(key3, 1, 1)
	// Test BITOP OR
	r.BITOP("OR", destKey, key4, key3)
	if r.BITCOUNT(destKey) != 3 {
		t.Errorf("BITOP OR: Expected BITCOUNT to be 3, got %d", r.BITCOUNT(destKey))
	}

	// Test BITOP XOR
	r.BITOP("XOR", destKey, key4, key3)
	if r.BITCOUNT(destKey) != 2 {
		t.Errorf("BITOP XOR: Expected BITCOUNT to be 2, got %d", r.BITCOUNT(destKey))
	}

	// Test BITOP NOT
	r.BITOP("NOT", destKey, key1)
	result, _ := r.Store[destKey].([]byte)
	if len(result)*8 != 16 { // Assuming default length of 64 bits
		t.Errorf("BITOP NOT: Expected result length to match key1, got %d bits", len(result)*8)
	}
}
