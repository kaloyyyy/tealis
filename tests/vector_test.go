package storage

import (
	"encoding/json"
	"os"
	"strings"
	"tealis/internal/storage"
	"testing"
)

func TestVectorSetAndGet(t *testing.T) {
	// Setup
	aofFilePath := "./snapshot"
	snapshotPath := "./snapshot"
	// Clean up the ./snapshot folder and files before starting the test

	defer os.Remove(aofFilePath) // Clean up the test AOF file

	// Initialize a RedisClone instance with AOF enabled
	r := storage.NewRedisClone(aofFilePath, snapshotPath, false)

	// Test VectorSet
	vector := []float64{1.0, 2.0, 3.0}
	response := r.VectorSet("vec1", vector)
	if response != "+OK\r\n" {
		t.Errorf("Expected +OK\\r\\n, got %s", response)
	}

	// Test VectorGet - existing vector
	response = r.VectorGet("vec1")
	var retrievedVector []float64
	err := json.Unmarshal([]byte(strings.Trim(response[4:], "\r\n")), &retrievedVector)
	if err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}
	if len(retrievedVector) != len(vector) {
		t.Errorf("Expected vector length %d, got %d", len(vector), len(retrievedVector))
	}
	for i, v := range vector {
		if retrievedVector[i] != v {
			t.Errorf("Expected value %f at index %d, got %f", v, i, retrievedVector[i])
		}
	}

	// Test VectorGet - non-existing key
	response = r.VectorGet("nonexistent")
	if response != "-ERR Key not found\r\n" {
		t.Errorf("Expected -ERR Key not found\\r\\n, got %s", response)
	}

	// Test VectorGet - key is not a vector
	r.Store["notavector"] = "string"
	response = r.VectorGet("notavector")
	if response != "-ERR Key is not a vector\r\n" {
		t.Errorf("Expected -ERR Key is not a vector\\r\\n, got %s", response)
	}
}

func TestVectorSearch(t *testing.T) {
	// Setup
	aofFilePath := "./snapshot"
	snapshotPath := "./snapshot"
	// Clean up the ./snapshot folder and files before starting the test

	defer os.Remove(aofFilePath) // Clean up the test AOF file

	// Initialize a RedisClone instance with AOF enabled
	r := storage.NewRedisClone(aofFilePath, snapshotPath, false)

	// Add vectors
	r.VectorSet("vec1", []float64{1.0, 0.0})
	r.VectorSet("vec2", []float64{0.0, 1.0})
	r.VectorSet("vec3", []float64{1.0, 1.0})
	r.VectorSet("vec4", []float64{0.0, 0.0})

	// Search with query vector
	query := []float64{1.0, 1.0}
	response := r.VectorSearch(query, 2)

	expectedKeys := []string{"vec3", "vec1"} // Closest two vectors
	for _, key := range expectedKeys {
		if !strings.Contains(response, key) {
			t.Errorf("Expected key %s in search results, got %s", key, response)
		}
	}

	// Test with k greater than available vectors
	response = r.VectorSearch(query, 10)
	expectedKeys = []string{"vec3", "vec1", "vec2", "vec4"}
	for _, key := range expectedKeys {
		if !strings.Contains(response, key) {
			t.Errorf("Expected key %s in search results, got %s", key, response)
		}
	}

	// Test with incompatible dimensions
	invalidQuery := []float64{1.0}
	response = r.VectorSearch(invalidQuery, 2)
	if !strings.Contains(response, "vec1") {
		t.Errorf("Expected fallback result with key vec4 for invalid query dimensions, got %s", response)
	}
}

func TestCosineSimilarity(t *testing.T) {
	a := []float64{1.0, 2.0, 3.0}
	b := []float64{1.0, 2.0, 3.0}
	c := []float64{0.0, 0.0, 0.0}

	// Test same vectors
	expected := 0.0 // Perfect match
	result := storage.CosineSimilarity(a, b)
	if result != expected {
		t.Errorf("Expected cosine similarity %f, got %f", expected, result)
	}

	// Test zero vector
	expected = 1.0
	result = storage.CosineSimilarity(a, c)
	if result != expected {
		t.Errorf("Expected cosine similarity %f, got %f", expected, result)
	}

	// Test different vectors with same magnitude expected cases<>
}
