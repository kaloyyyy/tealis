package storage

import (
	"encoding/json"
	"testing"
)

func TestUnmarshalAndMarshalJSON(t *testing.T) {
	// Original JSON data
	originalJSON := `{"key1": "value1", "nested": {"key2": "value2"}}`

	// Step 1: Unmarshal the JSON into a map
	var result map[string]interface{}
	err := json.Unmarshal([]byte(originalJSON), &result)
	if err != nil {
		t.Fatalf("Error unmarshaling JSON: %v", err)
	}

	// Validate structure after unmarshaling
	if result["key1"] != "value1" {
		t.Errorf("Expected key1 to be 'value1', got %v", result["key1"])
	}

	nested, ok := result["nested"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected 'nested' to be a map, got %T", result["nested"])
	}

	if nested["key2"] != "value2" {
		t.Errorf("Expected nested key2 to be 'value2', got %v", nested["key2"])
	}

	// Step 2: Marshal the data back to JSON
	marshaledJSON, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Error marshaling JSON: %v", err)
	}

	// Step 3: Unmarshal the marshaled JSON and compare with the original structure
	var reUnmarshaled map[string]interface{}
	err = json.Unmarshal(marshaledJSON, &reUnmarshaled)
	if err != nil {
		t.Fatalf("Error unmarshaling marshaled JSON: %v", err)
	}

	// Validate that the re-unmarshaled JSON matches the original structure
	if reUnmarshaled["key1"] != "value1" {
		t.Errorf("Expected re-unmarshaled key1 to be 'value1', got %v", reUnmarshaled["key1"])
	}

	reNested, ok := reUnmarshaled["nested"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected re-unmarshaled 'nested' to be a map, got %T", reUnmarshaled["nested"])
	}

	if reNested["key2"] != "value2" {
		t.Errorf("Expected re-unmarshaled nested key2 to be 'value2', got %v", reNested["key2"])
	}
}
