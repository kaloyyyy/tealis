package storage_test

import (
	"os"
	"reflect"
	"tealis/internal/storage"
	"testing"
)

func TestRedisCloneSortedSet(t *testing.T) {
	// Setup
	aofFilePath := "./snapshot"
	snapshotPath := "./snapshot"
	// Clean up the ./snapshot folder and files before starting the test

	defer os.Remove(aofFilePath) // Clean up the test AOF file

	// Initialize a RedisClone instance with AOF enabled
	r := storage.NewRedisClone(aofFilePath, snapshotPath, false)

	r.ZAdd("myzset", 1.0, "one")
	r.ZAdd("myzset", 2.0, "two")
	r.ZAdd("myzset", 3.0, "three")
	expectedZRange := []string{"one", "two", "three"}
	zrange := r.ZRange("myzset", 0, 2)
	if !reflect.DeepEqual(zrange, expectedZRange) {
		t.Fatalf("ZRange failed, expected %v, got %v", expectedZRange, zrange)
	}

	// Test ZRANK
	zrank := r.ZRank("myzset", "two")
	expectedRank := 1
	if zrank != expectedRank {
		t.Fatalf("ZRank failed, expected %d, got %d", expectedRank, zrank)
	}

	// Test ZREM
	removed := r.ZRem("myzset", "two")
	if !removed {
		t.Fatalf("ZRem failed, expected %v, got %v", true, removed)
	}
	expectedZRangeAfterRem := []string{"one", "three"}
	zrangeAfterRem := r.ZRange("myzset", 0, 2)
	if !reflect.DeepEqual(zrangeAfterRem, expectedZRangeAfterRem) {
		t.Fatalf("ZRange after ZRem failed, expected %v, got %v", expectedZRangeAfterRem, zrangeAfterRem)
	}

	// Test ZRANGEBYSCORE
	r.ZAdd("myzset", 2.5, "two-and-half")
	expectedRangeByScore := []string{"one", "two-and-half", "three"}
	zrangeByScore := r.ZRangeByScore("myzset", 1.0, 3.0)
	if !reflect.DeepEqual(zrangeByScore, expectedRangeByScore) {
		t.Fatalf("ZRangeByScore failed, expected %v, got %v", expectedRangeByScore, zrangeByScore)
	}

	// Test Non-existent Key
	zrangeNonExistent := r.ZRange("nonexistent", 0, 2)
	if zrangeNonExistent != nil {
		t.Fatalf("ZRange on nonexistent key failed, expected nil, got %v", zrangeNonExistent)
	}

	zrankNonExistent := r.ZRank("nonexistent", "key")
	if zrankNonExistent != -1 {
		t.Fatalf("ZRank on nonexistent key failed, expected -1, got %d", zrankNonExistent)
	}

	removedNonExistent := r.ZRem("nonexistent", "key")
	if removedNonExistent {
		t.Fatalf("ZRem on nonexistent key failed, expected false, got %v", removedNonExistent)
	}
}
