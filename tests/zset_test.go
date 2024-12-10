package storage_test

import (
	"tealis/internal/storage"
	"testing"
)

func TestSortedSetOperations(t *testing.T) {
	store := storage.NewRedisClone()

	// Test ZADD
	if added := store.ZADD("myzset", 1.0, "member1"); added != 1 {
		t.Errorf("Expected 1 for ZADD, got %d", added)
	}
	if added := store.ZADD("myzset", 2.0, "member2"); added != 1 {
		t.Errorf("Expected 1 for ZADD, got %d", added)
	}
	if added := store.ZADD("myzset", 3.0, "member1"); added != 0 {
		t.Errorf("Expected 0 for ZADD (update), got %d", added)
	}

	// Test ZRANGE
	members := store.ZRANGE("myzset", 0, -1)
	expected := []string{"member1", "member2"}
	if !equalStringSlices(members, expected) {
		t.Errorf("Expected ZRANGE result %v, got %v", expected, members)
	}

	// Test ZRANK
	if rank := store.ZRANK("myzset", "member1"); rank != 0 {
		t.Errorf("Expected rank 0 for member1, got %d", rank)
	}
	if rank := store.ZRANK("myzset", "member2"); rank != 1 {
		t.Errorf("Expected rank 1 for member2, got %d", rank)
	}
	if rank := store.ZRANK("myzset", "nonexistent"); rank != -1 {
		t.Errorf("Expected rank -1 for nonexistent member, got %d", rank)
	}

	// Test ZREM
	if removed := store.ZREM("myzset", "member2"); !removed {
		t.Error("Expected true for ZREM of member2, got false")
	}
	if removed := store.ZREM("myzset", "nonexistent"); removed {
		t.Error("Expected false for ZREM of nonexistent member, got true")
	}
	members = store.ZRANGE("myzset", 0, -1)
	expected = []string{"member1"}
	if !equalStringSlices(members, expected) {
		t.Errorf("Expected ZRANGE result %v, got %v", expected, members)
	}

	// Test ZRANGEBYSCORE
	store.ZADD("myzset", 2.0, "member2")
	store.ZADD("myzset", 3.5, "member3")
	members = store.ZRANGEBYSCORE("myzset", 2.0, 3.0)
	expected = []string{"member2"}
	if !equalStringSlices(members, expected) {
		t.Errorf("Expected ZRANGEBYSCORE result %v, got %v", expected, members)
	}
	members = store.ZRANGEBYSCORE("myzset", 0, 3.5)
	expected = []string{"member1", "member2", "member3"}
	if !equalStringSlices(members, expected) {
		t.Errorf("Expected ZRANGEBYSCORE result %v, got %v", expected, members)
	}
}

// Helper function to compare two string slices
func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
