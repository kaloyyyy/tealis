package storage_test

import (
	"reflect"
	"tealis/internal/storage"
	"testing"
	// Replace with the actual path to your RedisClone package
)

func TestJSONDel(t *testing.T) {
	r := storage.NewRedisClone() // Initialize your RedisClone instance

	// Test case 1: Delete a top-level key
	key := "testKey"
	data := `{"key1": "value1", "nested": {"key2": "value2", "key3": {"key4": "value4"}}, "array": [1, 2, 3]}`
	err := r.JSONSet(key, ".", data)
	if err != nil {
		t.Fatalf("JSONSet failed: %v", err)
	}

	// Delete a top-level key
	err = r.JSONDel(key, "key1")
	if err != nil {
		t.Fatalf("JSONDel failed: %v", err)
	}

	// Verify the top-level key is deleted
	expected := map[string]interface{}{
		"nested": map[string]interface{}{
			"key2": "value2",
			"key3": map[string]interface{}{
				"key4": "value4",
			},
		},
		"array": []interface{}{1, 2, 3},
	}

	result, err := r.JSONGet(key, ".")
	if err != nil {
		t.Fatalf("JSONGet failed: %v", err)
	}

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Unexpected result for top-level delete: got %v, want %v", result, expected)
	}

	// Test case 2: Delete a nested key
	err = r.JSONDel(key, "nested.key2")
	if err != nil {
		t.Fatalf("JSONDel failed: %v", err)
	}

	// Verify the nested key is deleted
	expected = map[string]interface{}{
		"nested": map[string]interface{}{
			"key3": map[string]interface{}{
				"key4": "value4",
			},
		},
		"array": []interface{}{1, 2, 3},
	}

	result, err = r.JSONGet(key, ".")
	if err != nil {
		t.Fatalf("JSONGet failed: %v", err)
	}

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Unexpected result for nested delete: got %v, want %v", result, expected)
	}

	// Test case 3: Delete an array element
	err = r.JSONDel(key, "array.[1]") // Deletes the second element (index 1)
	if err != nil {
		t.Fatalf("JSONDel failed: %v", err)
	}

	// Verify the array element is deleted
	expected = map[string]interface{}{
		"nested": map[string]interface{}{
			"key3": map[string]interface{}{
				"key4": "value4",
			},
		},
		"array": []interface{}{1, 3},
	}

	result, err = r.JSONGet(key, ".")
	if err != nil {
		t.Fatalf("JSONGet failed: %v", err)
	}

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Unexpected result for array element delete: got %v, want %v", result, expected)
	}

	// Test case 4: Delete the entire key
	err = r.JSONDel(key, ".")
	if err != nil {
		t.Fatalf("JSONDel failed for root path: %v", err)
	}

	// Verify the key is completely removed
	_, err = r.JSONGet(key, ".")
	if err == nil {
		t.Errorf("Expected key to be deleted, but it still exists")
	}
}
