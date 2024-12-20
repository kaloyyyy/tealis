package storage

import (
	"os"
	"reflect"
	"tealis/internal/storage"
	"testing"
)

func TestJSONDel(t *testing.T) {
	// Setup
	aofFilePath := "./snapshot"
	snapshotPath := "./snapshot"
	// Clean up the ./snapshot folder and files before starting the test

	defer os.Remove(aofFilePath) // Clean up the test AOF file

	// Initialize a Tealis instance with AOF enabled
	r := storage.NewTealis(aofFilePath, snapshotPath, false)

	key := "testKey"
	data := `{"key1": "value1", "nested": {"key2": "value2"}}`
	err := r.JSONSet(key, ".", data)
	if err != nil {
		t.Fatalf("JSONSet failed: %v", err)
	}

	// Delete a nested key
	err = r.JSONDel(key, "nested.key2")
	if err != nil {
		t.Fatalf("JSONDel failed: %v", err)
	}

	// Verify deletion
	result, err := r.JSONGet(key, ".")
	if err != nil {
		t.Fatalf("JSONGet failed: %v", err)
	}

	expected := "{}"
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Unexpected result: got %v, want %v", result, expected)
	}
}
