package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	_ "reflect"
	"strconv"
	"strings"
)

// JSONSet JSON.SET function (sets a value at a path in a JSON-like structure)
func (r *RedisClone) JSONSet(key string, path string, value string) error {
	r.Mu.Lock()
	defer r.Mu.Unlock()

	// Handle the special case where the path is "."

	// Step 1: Unmarshal the JSON into a map
	var result map[string]interface{}
	err := json.Unmarshal([]byte(value), &result)
	if err != nil {
		fmt.Printf("Error unmarshaling JSON: %v", err)
	}
	// Directly serialize the value as JSON and store it
	serializedData, err := json.Marshal(result)
	if err != nil {
		return err
	}
	var jsonData map[string]interface{}
	if err := json.Unmarshal(serializedData, &jsonData); err != nil {
		return err
	}
	fmt.Printf("unserial %s", jsonData)
	r.Store[key] = result
	return nil
}

// JSONGet retrieves a value from a JSON-like structure
func (r *RedisClone) JSONGet(key string, path string) (interface{}, error) {

	// Retrieve the raw JSON data from the store
	existing, exists := r.Store[key]
	if !exists {
		return nil, fmt.Errorf("key not found")
	}

	// Unmarshal the stored JSON string into a generic interface
	var jsonData interface{}
	//if err := json.Unmarshal([]byte(existing), &jsonData); err != nil {
	//	return nil, fmt.Errorf("failed to parse stored JSON: %v", err)
	//}
	jsonData = existing
	// Handle the special case for root path "."
	if path == "." {
		return jsonData, nil
	}

	// Pass the path as-is to getAtPath to handle nested paths
	return getAtPath(jsonData, path)
}

// JSONDel JSON.DEL function (deletes a value from a JSON-like structure)
func (r *RedisClone) JSONDel(key string, path string) error {
	// Remove leading dot if present
	data, err := r.JSONGet(key, ".")
	if err != nil {
		return err
	}
	if strings.HasPrefix(path, ".") {
		path = path[1:]
	}

	// Split the path into parts
	parts := strings.Split(path, ".")

	// If the path is empty or only contains "." delete the whole data
	if path == "" || path == "." {
		if _, exists := r.Store[key]; exists {
			delete(r.Store, key)
			delete(r.Expiries, key)
			return nil
		}
		return nil
	}

	// Traverse the data based on the parts
	currentValue := data
	var done = false
	for i, part := range parts {
		// Check if current value is a map
		if m, ok := currentValue.(map[string]interface{}); ok {
			if i == len(parts)-1 {
				// If we're at the last part, delete the key
				delete(m, part)
				done = true
				break
			}
			currentValue = m[part]
			if currentValue == nil {
				return fmt.Errorf("key '%s' not found", part)
			}
		} else if arr, ok := currentValue.([]interface{}); ok {
			// If current value is an array, try to access the index (part should be numeric)
			var index int
			_, err := fmt.Sscanf(part, "%d", &index)
			if err != nil || index < 0 || index >= len(arr) {
				return fmt.Errorf("invalid index '%s' for array", part)
			}
			if i == len(parts)-1 {
				// If we're at the last part, set the element to nil (deletes it)

				arr[index] = nil
				done = true
				break
			}
			currentValue = arr[index]
		} else {
			return fmt.Errorf("invalid path '%s' at part '%s'", path, part)
		}
	}
	serializedData, err := json.Marshal(currentValue)
	if err != nil {
		return err
	}
	r.Store[key] = string(serializedData)
	if done {
		return nil
	}
	return fmt.Errorf("path '%s' not found", path)
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

// getAtPath retrieves the value from a map based on the given dot-separated path.
func getAtPath(data interface{}, path string) (interface{}, error) {
	// Remove leading dot if present
	if strings.HasPrefix(path, ".") {
		path = path[1:]
	}

	// Split the path into parts
	parts := strings.Split(path, ".")

	// Traverse the data based on the parts
	currentValue := data
	for _, part := range parts {
		// Check if current value is a map
		if m, ok := currentValue.(map[string]interface{}); ok {
			currentValue = m[part]
			if currentValue == nil {
				return nil, fmt.Errorf("key '%s' not found", part)
			}
		} else if arr, ok := currentValue.([]interface{}); ok {
			// If current value is an array, try to access the index (part should be numeric)
			var index int
			_, err := fmt.Sscanf(part, "%d", &index)
			if err != nil || index < 0 || index >= len(arr) {
				return nil, fmt.Errorf("invalid index '%s' for array", part)
			}
			currentValue = arr[index]
		} else {
			// If the current value is neither a map nor an array, return an error
			return nil, fmt.Errorf("invalid path '%s' at part '%s'", path, part)
		}
	}
	return currentValue, nil
}

// JSONArrAppend appends values to an array at the specified path.
func (r *RedisClone) JSONArrAppend(key string, path string, values []interface{}) error {
	r.Mu.Lock()
	defer r.Mu.Unlock()
	// Remove leading dot if present
	if strings.HasPrefix(path, ".") {
		path = path[1:]
	}
	data, err := r.JSONGet(key, ".")
	if err != nil {
		return err
	}
	// Split the path into parts
	parts := strings.Split(path, ".")
	done := false
	// Traverse the data based on the parts
	currentValue := data
	for i, part := range parts {
		// If we're at the last part, ensure it's an array and append to it
		if i == len(parts)-1 {
			if m, ok := currentValue.(map[string]interface{}); ok {
				if arr, ok := m[part].([]interface{}); ok {
					m[part] = append(arr, values...)
					done = true
					break
				}
				return fmt.Errorf("key '%s' is not an array", part)
			}
			return fmt.Errorf("path '%s' is not a valid map", path)
		}

		// Traverse deeper into the structure
		if m, ok := currentValue.(map[string]interface{}); ok {
			if next, exists := m[part]; exists {
				currentValue = next
			} else {
				return fmt.Errorf("key '%s' not found", part)
			}
		} else {
			return fmt.Errorf("path '%s' is not a valid map at '%s'", path, part)
		}
	}
	serializedData, err := json.Marshal(currentValue)
	if err != nil {
		return err
	}
	r.Store[key] = string(serializedData)
	if done {
		return nil
	}
	return fmt.Errorf("invalid path '%s'", path)
}
