package storage

import (
	"os"
	"sort"
	"tealis/internal/storage"
	"testing"
)

func TestRedisCloneSetOperations(t *testing.T) {
	// Setup
	aofFilePath := "./snapshot"
	snapshotPath := "./snapshot"
	// Clean up the ./snapshot folder and files before starting the test

	defer os.Remove(aofFilePath) // Clean up the test AOF file

	// Initialize a Tealis instance with AOF enabled
	r := storage.NewTealis(aofFilePath, snapshotPath, false)

	// Test SADD
	t.Run("SADD", func(t *testing.T) {
		length := r.SADD("myset", "a", "b", "c", "d")
		if length != 4 {
			t.Errorf("Expected set length 4, but got %d", length)
		}

		// Verify the set content
		members := r.SMEMBERS("myset")
		expectedMembers := []string{"a", "b", "c", "d"}
		sort.Strings(expectedMembers)
		sort.Strings(members)
		if !equal(members, expectedMembers) {
			t.Errorf("Expected set %v, but got %v", expectedMembers, members)
		}
	})

	// Test SREM
	t.Run("SREM", func(t *testing.T) {
		removedCount := r.SREM("myset", "a", "b")
		if removedCount != 2 {
			t.Errorf("Expected to remove 2 members, but removed %d", removedCount)
		}

		// Verify the set content after removal
		members := r.SMEMBERS("myset")
		expectedMembers := []string{"c", "d"}
		sort.Strings(members)
		sort.Strings(expectedMembers)
		if !equal(members, expectedMembers) {
			t.Errorf("Expected set %v, but got %v", expectedMembers, members)
		}
	})

	// Test SISMEMBER
	t.Run("SISMEMBER", func(t *testing.T) {
		exists := r.SISMEMBER("myset", "c")
		if !exists {
			t.Error("Expected member 'c' to exist in the set, but it does not")
		}

		exists = r.SISMEMBER("myset", "a")
		if exists {
			t.Error("Expected member 'a' to not exist in the set, but it does")
		}
	})

	// Test SUNION
	t.Run("SUNION", func(t *testing.T) {
		// Add more sets
		r.SADD("myset2", "e", "f", "g")
		r.SADD("myset3", "h", "i")

		union := r.SUNION("myset", "myset2", "myset3")
		expectedUnion := []string{"d", "e", "f", "g", "h", "i", "c"}
		sort.Strings(union)
		sort.Strings(expectedUnion)
		if !equal(union, expectedUnion) {
			t.Errorf("Expected union %v, but got %v", expectedUnion, union)
		}
	})

	// Test SINTER
	t.Run("SINTER", func(t *testing.T) {
		// Add some common members to test intersection
		r.SADD("myset2", "c", "d", "h")

		intersection := r.SINTER("myset", "myset2")
		expectedIntersection := []string{"c", "d"}
		sort.Strings(expectedIntersection)
		sort.Strings(intersection)
		if !equal(intersection, expectedIntersection) {
			t.Errorf("Expected intersection %v, but got %v", expectedIntersection, intersection)
		}
	})

	// Test SDIFF
	t.Run("SDIFF", func(t *testing.T) {
		// Add some different members to test difference
		r.SADD("myset2", "e", "f", "g")

		difference := r.SDIFF("myset", "myset2")
		var expectedDifference []string
		if !equal(difference, expectedDifference) {
			t.Errorf("Expected difference %v, but got %v", expectedDifference, difference)
		}
	})
}
