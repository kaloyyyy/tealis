package protocol

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

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
	parts := strings.Split(path, ".")
	return getValue(data, parts)
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
