package storage_test

import (
	"os"
	"reflect"
	"tealis/internal/storage"
	"testing"
)

func TestRedisCloneSortedSet(t *testing.T) {
	aofFilePath := "test.aof"
	defer os.Remove(aofFilePath) // Clean up the test AOF file

	// Initialize a RedisClone instance with AOF enabled
	store := storage.NewRedisClone(aofFilePath, "", true)
	// Test ZADD
	store.ZAdd("myzset", 1.0, "one")
	store.ZAdd("myzset", 2.0, "two")
	store.ZAdd("myzset", 3.0, "three")
	expectedZRange := []string{"one", "two", "three"}
	zrange := store.ZRange("myzset", 0, 2)
	if !reflect.DeepEqual(zrange, expectedZRange) {
		t.Fatalf("ZRange failed, expected %v, got %v", expectedZRange, zrange)
	}

	// Test ZRANK
	zrank := store.ZRank("myzset", "two")
	expectedRank := 1
	if zrank != expectedRank {
		t.Fatalf("ZRank failed, expected %d, got %d", expectedRank, zrank)
	}

	// Test ZREM
	removed := store.ZRem("myzset", "two")
	if !removed {
		t.Fatalf("ZRem failed, expected %v, got %v", true, removed)
	}
	expectedZRangeAfterRem := []string{"one", "three"}
	zrangeAfterRem := store.ZRange("myzset", 0, 2)
	if !reflect.DeepEqual(zrangeAfterRem, expectedZRangeAfterRem) {
		t.Fatalf("ZRange after ZRem failed, expected %v, got %v", expectedZRangeAfterRem, zrangeAfterRem)
	}

	// Test ZRANGEBYSCORE
	store.ZAdd("myzset", 2.5, "two-and-half")
	expectedRangeByScore := []string{"one", "two-and-half", "three"}
	zrangeByScore := store.ZRangeByScore("myzset", 1.0, 3.0)
	if !reflect.DeepEqual(zrangeByScore, expectedRangeByScore) {
		t.Fatalf("ZRangeByScore failed, expected %v, got %v", expectedRangeByScore, zrangeByScore)
	}

	// Test Non-existent Key
	zrangeNonExistent := store.ZRange("nonexistent", 0, 2)
	if zrangeNonExistent != nil {
		t.Fatalf("ZRange on nonexistent key failed, expected nil, got %v", zrangeNonExistent)
	}

	zrankNonExistent := store.ZRank("nonexistent", "key")
	if zrankNonExistent != -1 {
		t.Fatalf("ZRank on nonexistent key failed, expected -1, got %d", zrankNonExistent)
	}

	removedNonExistent := store.ZRem("nonexistent", "key")
	if removedNonExistent {
		t.Fatalf("ZRem on nonexistent key failed, expected false, got %v", removedNonExistent)
	}
}
