package storage

import (
	"os"
	"tealis/internal/storage"
	"testing"
)

func TestRedisCloneListOperations(t *testing.T) {
	// Create a new Redis clone instance
	aofFilePath := "test.aof"
	defer os.Remove(aofFilePath) // Clean up the test AOF file

	// Initialize a RedisClone instance with AOF enabled
	r := storage.NewRedisClone(aofFilePath, "./snapshot", true)
	// Test RPUSH
	t.Run("RPUSH", func(t *testing.T) {
		// Add items to the list
		length := r.RPUSH("mylist", "a", "b", "c")
		if length != 3 {
			t.Errorf("Expected list length 3, but got %d", length)
		}

		// Verify the list content
		list := r.LRANGE("mylist", 0, -1)
		expectedList := []string{"a", "b", "c"}
		if !equal(list, expectedList) {
			t.Errorf("Expected list %v, but got %v", expectedList, list)
		}
	})

	// Test LPUSH
	t.Run("LPUSH", func(t *testing.T) {
		// Add items to the front of the list
		length := r.LPUSH("mylist", "x", "y")
		if length != 5 {
			t.Errorf("Expected list length 5, but got %d", length)
		}

		// Verify the list content
		list := r.LRANGE("mylist", 0, -1)
		expectedList := []string{"x", "y", "a", "b", "c"}
		if !equal(list, expectedList) {
			t.Errorf("Expected list %v, but got %v", expectedList, list)
		}
	})

	// Test LPOP
	t.Run("LPOP", func(t *testing.T) {
		value, exists := r.LPOP("mylist")
		if !exists {
			t.Error("Expected LPOP to return true, but got false")
		}
		if value != "x" {
			t.Errorf("Expected 'x', but got %s", value)
		}

		// Verify the list content after LPOP
		list := r.LRANGE("mylist", 0, -1)
		expectedList := []string{"y", "a", "b", "c"}
		if !equal(list, expectedList) {
			t.Errorf("Expected list %v, but got %v", expectedList, list)
		}
	})

	// Test RPOP
	t.Run("RPOP", func(t *testing.T) {
		value, exists := r.RPOP("mylist")
		if !exists {
			t.Error("Expected RPOP to return true, but got false")
		}
		if value != "c" {
			t.Errorf("Expected 'c', but got %s", value)
		}

		// Verify the list content after RPOP
		list := r.LRANGE("mylist", 0, -1)
		expectedList := []string{"y", "a", "b"}
		if !equal(list, expectedList) {
			t.Errorf("Expected list %v, but got %v", expectedList, list)
		}
	})

	// Test LRANGE
	t.Run("LRANGE", func(t *testing.T) {
		// Retrieve a range of elements
		list := r.LRANGE("mylist", 0, 1)
		expectedList := []string{"y", "a"}
		if !equal(list, expectedList) {
			t.Errorf("Expected list %v, but got %v", expectedList, list)
		}
	})
}

// Helper function to check if two slices are equal
func equal(a, b []string) bool {
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
