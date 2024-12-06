package storage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"
)

type RedisClone struct {
	mu       sync.RWMutex
	Store    map[string]string
	expiries map[string]time.Time
}

func NewRedisClone() *RedisClone {
	return &RedisClone{
		Store:    make(map[string]string),
		expiries: make(map[string]time.Time),
	}
}

// Set saves a key-value pair with an optional TTL.
func (r *RedisClone) Set(key, value string, ttl time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.Store[key] = value
	if ttl > 0 {
		r.expiries[key] = time.Now().Add(ttl)
	} else {
		delete(r.expiries, key)
	}
}

// Get retrieves the value for a key.
func (r *RedisClone) Get(key string) (string, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if expiry, exists := r.expiries[key]; exists && time.Now().After(expiry) {
		r.mu.Lock()
		delete(r.Store, key)
		delete(r.expiries, key)
		r.mu.Unlock()
		return "", false
	}

	value, exists := r.Store[key]
	return value, exists
}

// Del deletes a key from the store.
func (r *RedisClone) Del(key string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.Store[key]; exists {
		delete(r.Store, key)
		delete(r.expiries, key)
		return true
	}
	return false
}

// Exists checks if a key exists in the store.
func (r *RedisClone) Exists(key string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, exists := r.Store[key]
	return exists
}

// StartCleanup periodically cleans expired keys.
func (r *RedisClone) StartCleanup(ctx context.Context) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Second):
				now := time.Now()
				r.mu.Lock()
				for key, expiry := range r.expiries {
					if now.After(expiry) {
						delete(r.Store, key)
						delete(r.expiries, key)
					}
				}
				r.mu.Unlock()
			}
		}
	}()
}

// Append appends a value to an existing key.
func (r *RedisClone) Append(key, value string) int {
	r.mu.Lock()
	defer r.mu.Unlock()

	if current, exists := r.Store[key]; exists {
		r.Store[key] = current + value
	} else {
		r.Store[key] = value
	}
	return len(r.Store[key])
}

// StrLen returns the length of a string value for a key.
func (r *RedisClone) StrLen(key string) int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if value, exists := r.Store[key]; exists {
		return len(value)
	}
	return 0
}

// IncrBy increments a key by a specified value.
func (r *RedisClone) IncrBy(key string, increment int) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	current, exists := r.Store[key]
	if !exists {
		r.Store[key] = strconv.Itoa(increment)
		return increment, nil
	}

	currentInt, err := strconv.Atoi(current)
	if err != nil {
		return 0, fmt.Errorf("value is not an integer")
	}

	newValue := currentInt + increment
	r.Store[key] = strconv.Itoa(newValue)
	return newValue, nil
}

// GetRange retrieves a substring from a value.
func (r *RedisClone) GetRange(key string, start, end int) string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	value, exists := r.Store[key]
	if !exists {
		return ""
	}

	if start < 0 {
		start = len(value) + start
	}
	if end < 0 {
		end = len(value) + end
	}

	if start < 0 {
		start = 0
	}
	if start >= len(value) {
		return ""
	}

	if end >= len(value) {
		end = len(value) - 1
	}

	if start > end {
		return ""
	}

	return value[start : end+1]
}

// SetRange sets a substring at the specified offset.
func (r *RedisClone) SetRange(key string, offset int, value string) int {
	r.mu.Lock()
	defer r.mu.Unlock()

	currentValue, exists := r.Store[key]
	if !exists {
		currentValue = ""
	}

	if offset > len(currentValue) {
		currentValue = currentValue + strings.Repeat(" ", offset-len(currentValue))
	}

	newValue := currentValue[:offset] + value
	r.Store[key] = newValue
	return len(newValue)
}

// Keys returns keys that match a pattern.
func (r *RedisClone) Keys(pattern string) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var matchedKeys []string
	for key := range r.Store {
		if pattern == "*" || matchesPattern(key, pattern) {
			matchedKeys = append(matchedKeys, key)
		}
	}
	return matchedKeys
}

func matchesPattern(key, pattern string) bool {
	matched, err := path.Match(pattern, key)
	return err == nil && matched
}

// JSONSet JSON.SET function (sets a value at a path in a JSON-like structure)
func (r *RedisClone) JSONSet(key string, path string, value interface{}) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	var data map[string]interface{}
	if existing, exists := r.Store[key]; exists {
		if err := json.Unmarshal([]byte(existing), &data); err != nil {
			return err
		}
	} else {
		data = make(map[string]interface{})
	}

	updatedData, err := setAtPath(data, path, value)
	if err != nil {
		return err
	}

	serializedData, err := json.Marshal(updatedData)
	if err != nil {
		return err
	}
	fmt.Printf("set json k:%s p: %s v: %s\n", key, path, value)
	fmt.Printf("set json data: %s up: %s sd: %s \n", data, updatedData, serializedData)
	r.Store[key] = string(serializedData)
	return nil
}

// JSONGet retrieves a value from a JSON-like structure
func (r *RedisClone) JSONGet(key, path string) (interface{}, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Debugging output to help track the process
	fmt.Printf("json get %s path %s\n", key, path)

	// Retrieve the stored data
	data, exists := r.Store[key]
	if !exists {
		return nil, errors.New("key not found")
	}

	// Deserialize the stored JSON data into a map
	var jsonData map[string]interface{}
	if err := json.Unmarshal([]byte(data), &jsonData); err != nil {
		return nil, err
	}

	// Call getAtPath to retrieve the value at the specified path
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

func getAtPath(data map[string]interface{}, path string) (interface{}, error) {
	// If path is just ".", return the whole object
	if path == "." {
		return data, nil
	}

	// Remove the leading dot from the path
	path = strings.TrimPrefix(path, ".")

	// Split the path by "." to handle nested paths
	parts := strings.Split(path, ".")

	// Traverse the JSON structure by iterating through the parts
	var current interface{} = data
	for _, part := range parts {
		switch current := current.(type) {
		case map[string]interface{}:
			// If it's a map, look for the key
			if value, exists := current[part]; exists {
				var newVal = value.(map[string]interface{})
				current = newVal
			} else {
				return nil, fmt.Errorf("path not found: %s", part)
			}
		case []interface{}:
			// If it's an array, we need to handle indices
			index, err := parseIndex(part)
			if err != nil || index >= len(current) {
				return nil, fmt.Errorf("invalid array index: %s", part)
			}
			current[0] = current[index]
		default:
			return nil, fmt.Errorf("path not found or invalid type at: %s", part)
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
