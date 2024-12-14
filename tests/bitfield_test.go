package storage

import (
	"fmt"
	"tealis/internal/storage"
	"testing"
)

func TestBITFIELD(t *testing.T) {
	redis := storage.NewRedisClone() // Assume NewRedisClone initializes your Redis clone.

	// Test SET and GET
	var list []int = []int{0, 100}

	_, err := redis.BITFIELD("SET", "key", "i32", list)
	if err != nil {
		t.Fatalf("Failed to SET: %v", err)
	}
	list = []int{0}

	value, err := redis.BITFIELD("GET", "key", "i32", list)
	if err != nil {
		t.Fatalf("Failed to GET: %v", err)
	}
	if value[0] != int32(100) {
		t.Fatalf("Expected 100, got %v", value[0])
	}

	// Test INCRBY
	list = []int{0, 10}

	value, err = redis.BITFIELD("INCRBY", "key", "i32", list)
	if err != nil {
		t.Fatalf("Failed to INCRBY: %v", err)
	}
	if value[0] != int64(110) {
		t.Fatalf("Expected 110, got %v", value[0])
	}

	// Test boundary condition (dynamic resizing)
	list = []int{0, 500}

	_, err = redis.BITFIELD("SET", "key", "i64", list)
	if err != nil {
		t.Fatalf("Failed to SET with resizing: %v", err)
	}
	list = []int{0}
	value, err = redis.BITFIELD("GET", "key", "i64", list)
	if err != nil {
		t.Fatalf("Failed to GET after resizing: %v", err)
	}
	if value[0] != int64(500) {
		t.Fatalf("Expected 500, got %v", value[0])
	}

	// Test INCRBY on newly set value
	list = []int{64, -200}
	value, err = redis.BITFIELD("INCRBY", "key", "i64", list)
	if err != nil {
		t.Fatalf("Failed to INCRBY: %v", err)
	}
	if value[0] != int64(300) {
		t.Fatalf("Expected 300, got %v", value[0])
	}

	list = []int{0, 100}
	// Test error handling for invalid type
	_, err = redis.BITFIELD("SET", "key", "invalid", list)
	if err == nil || err.Error() != "-ERR invalid bit type invalid" {
		t.Fatalf("Expected invalid bit type error, got %v", err)
	}

	// Test error handling for invalid operation
	list = []int{0, 100}
	_, err = redis.BITFIELD("INVALID_OP", "key", "i32", list)
	if err == nil || err.Error() != "-ERR unknown BITFIELD operation INVALID_OP" {
		t.Fatalf("Expected unknown operation error, got %v", err)
	}

	// Test error handling for out-of-range offset
	_, err = redis.BITFIELD("GET", "key", "i32", []int{1000000})
	if err == nil || err.Error() != "-ERR offset out of range" {
		t.Fatalf("Expected offset out of range error, got %v", err)
	}

	fmt.Println("All BITFIELD tests passed!")
}
