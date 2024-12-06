package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// JSONSet JSON.SET function (sets a value at a path in a JSON-like structure)
func (r *RedisClone) JSONSet(key string, path string, value interface{}) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Handle the special case where the path is "."
	if path == "." {
		// Directly serialize the value as JSON and store it
		serializedData, err := json.Marshal(value)
		if err != nil {
			return err
		}
		r.Store[key] = string(serializedData)
		return nil
	}

	// Otherwise, handle nested paths
	var data map[string]interface{}
	if existing, exists := r.Store[key]; exists {
		// Unmarshal the existing data into a map
		data = make(map[string]interface{})
		if err := json.Unmarshal([]byte(existing), &data); err != nil {
			return err
		}
	} else {
		data = make(map[string]interface{})
	}

	// Set the value at the specified path
	updatedData, err := setAtPath(data, path, value)
	if err != nil {
		return err
	}

	// Serialize the updated data and store it
	serializedData, err := json.Marshal(updatedData)
	if err != nil {
		return err
	}
	r.Store[key] = string(serializedData)
	return nil
}

// JSONGet retrieves a value from a JSON-like structure
func (r *RedisClone) JSONGet(key string, path string) (interface{}, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Retrieve the raw JSON data from the store
	existing, exists := r.Store[key]
	if !exists {
		return nil, fmt.Errorf("key not found")
	}

	// Unmarshal the stored JSON string into a generic interface
	var jsonData interface{}
	if err := json.Unmarshal([]byte(existing), &jsonData); err != nil {
		return nil, fmt.Errorf("failed to parse stored JSON: %v", err)
	}

	// Handle the special case for root path "."
	if path == "." {
		return jsonData, nil
	}

	// Pass the path as-is to getAtPath to handle nested paths
	return getAtPath(jsonData, path)
}

// JSONDel JSON.DEL function (deletes a value from a JSON-like structure)
func (r *RedisClone) JSONDel(key, path string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	data, exists := r.Store[key]
	if !exists {
		return errors.New("key not found")
	}

	var jsonData map[string]interface{}
	if err := json.Unmarshal([]byte(data), &jsonData); err != nil {
		return err
	}

	updatedData, err := deleteAtPath(jsonData, path)
	if err != nil {
		return err
	}

	serializedData, err := json.Marshal(updatedData)
	if err != nil {
		return err
	}

	r.Store[key] = string(serializedData)
	return nil
}

// JSONArrAppend JSON.ARRAPPEND function (appends to an array in a JSON-like structure)
func (r *RedisClone) JSONArrAppend(key, path string, values ...interface{}) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	data, exists := r.Store[key]
	if !exists {
		return errors.New("key not found")
	}

	var jsonData map[string]interface{}
	if err := json.Unmarshal([]byte(data), &jsonData); err != nil {
		return err
	}

	updatedData, err := appendToArray(jsonData, path, values...)
	if err != nil {
		return err
	}

	serializedData, err := json.Marshal(updatedData)
	if err != nil {
		return err
	}

	r.Store[key] = string(serializedData)
	return nil
}

func setAtPath(data interface{}, path string, value interface{}) (interface{}, error) {
	parts := strings.Split(path, ".")
	return setValue(data, parts, value)
}

func setValue(data interface{}, parts []string, value interface{}) (interface{}, error) {
	if len(parts) == 0 {
		return value, nil
	}

	// If we're dealing with a map (JSON object)
	if m, ok := data.(map[string]interface{}); ok {
		key := parts[0]
		rest := parts[1:]

		// Create nested maps if they don't exist
		if len(rest) > 0 {
			if _, exists := m[key]; !exists {
				m[key] = make(map[string]interface{})
			}
			// Recursively navigate the map
			updatedValue, err := setValue(m[key], rest, value)
			if err != nil {
				return nil, err
			}
			m[key] = updatedValue
		} else {
			// Set the value at the key
			m[key] = value
		}
		return m, nil
	}

	// If we're dealing with a slice (JSON array)
	if arr, ok := data.([]interface{}); ok {
		// For simplicity, we're assuming arrays are addressed by index
		if len(parts) == 1 {
			index, err := parseIndex(parts[0])
			if err != nil || index >= len(arr) {
				return nil, errors.New("invalid array index")
			}
			arr[index] = value
			return arr, nil
		}
	}

	return nil, errors.New("invalid path or data type")
}

// Helper to parse the index from the path
func parseIndex(part string) (int, error) {
	if strings.HasPrefix(part, "[") && strings.HasSuffix(part, "]") {
		// Remove the square brackets and parse the number
		indexStr := part[1 : len(part)-1]
		return strconv.Atoi(indexStr)
	}
	return 0, fmt.Errorf("invalid array index: %s", part)
}
func getAtPath(data interface{}, path string) (interface{}, error) {
	// If path is ".", return the entire object
	if path == "." {
		return data, nil
	}

	// Remove the leading dot and split the path into parts
	parts := strings.Split(strings.TrimPrefix(path, "."), ".")

	// Traverse the JSON structure
	var current interface{} = data
	var dataStr = data
	fmt.Printf("datastr %s", dataStr)
	err := json.Unmarshal([]byte(dataStr.(string)), &current)
	if err != nil {
		fmt.Println("Error unmarshaling JSON:", err)
		return nil, nil
	}

	for _, part := range parts {
		// Check the type of current, and navigate accordingly
		v := reflect.TypeOf(current)
		switch v.String() {
		case "map[string]interface {}":
			// Look for the key in the map
			val, exists := current.(map[string]interface{})[part]
			if !exists {
				return nil, fmt.Errorf("path not found: %s", part)
			}
			current = val
		case "[]interface{}":
			// Handle array indexing (not currently supported in your example)
			return nil, fmt.Errorf("unexpected array at path: %s", part)
		default:
			return nil, fmt.Errorf("unexpected type at path %s: %T", part, current)
		}
	}

	return current, nil
}

func getValue(data interface{}, parts []string) (interface{}, error) {
	if len(parts) == 0 {
		return data, nil
	}

	// If it's a map (JSON object)
	if m, ok := data.(map[string]interface{}); ok {
		key := parts[0]
		rest := parts[1:]

		if val, exists := m[key]; exists {
			return getValue(val, rest)
		}
		return nil, errors.New("key not found")
	}

	// If it's a slice (JSON array)
	if arr, ok := data.([]interface{}); ok {
		index, err := parseIndex(parts[0])
		if err != nil || index >= len(arr) {
			return nil, errors.New("invalid array index")
		}
		return getValue(arr[index], parts[1:])
	}

	return nil, errors.New("invalid path or data type")
}

func deleteAtPath(data interface{}, path string) (interface{}, error) {
	parts := strings.Split(path, ".")
	return deleteValue(data, parts)
}

func deleteValue(data interface{}, parts []string) (interface{}, error) {
	if len(parts) == 0 {
		return nil, nil // Return nil to delete the value
	}

	// If it's a map (JSON object)
	if m, ok := data.(map[string]interface{}); ok {
		key := parts[0]
		rest := parts[1:]

		if len(rest) == 0 {
			delete(m, key) // Delete the key from the map
			return m, nil
		}

		if val, exists := m[key]; exists {
			updatedVal, err := deleteValue(val, rest)
			if err != nil {
				return nil, err
			}
			m[key] = updatedVal
			return m, nil
		}
		return nil, errors.New("key not found")
	}

	// If it's a slice (JSON array)
	if arr, ok := data.([]interface{}); ok {
		index, err := parseIndex(parts[0])
		if err != nil || index >= len(arr) {
			return nil, errors.New("invalid array index")
		}
		// Remove the element from the array
		arr = append(arr[:index], arr[index+1:]...)
		return arr, nil
	}

	return nil, errors.New("invalid path or data type")
}

func appendToArray(data interface{}, path string, values ...interface{}) (interface{}, error) {
	parts := strings.Split(path, ".")
	return appendValues(data, parts, values)
}

func appendValues(data interface{}, parts []string, values []interface{}) (interface{}, error) {
	if len(parts) == 0 {
		// If we're at the right path, append to the array
		if arr, ok := data.([]interface{}); ok {
			arr = append(arr, values...)
			return arr, nil
		}
		return nil, errors.New("path does not point to an array")
	}

	// If it's a map (JSON object)
	if m, ok := data.(map[string]interface{}); ok {
		key := parts[0]
		rest := parts[1:]

		if val, exists := m[key]; exists {
			updatedVal, err := appendValues(val, rest, values)
			if err != nil {
				return nil, err
			}
			m[key] = updatedVal
			return m, nil
		}
	}

	return nil, errors.New("invalid path or data type")
}
