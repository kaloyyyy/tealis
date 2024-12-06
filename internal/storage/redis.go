package storage

import (
	"context"
	"encoding/json"
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

func (r *RedisClone) StartCleanup(ctx context.Context) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				// Stop the cleanup goroutine if the context is canceled
				return
			case <-time.After(time.Second):
				// Run cleanup every second
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

func (r *RedisClone) StrLen(key string) int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if value, exists := r.Store[key]; exists {
		return len(value)
	}
	return 0
}
func (r *RedisClone) IncrBy(key string, increment int) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	current, exists := r.Store[key]
	if !exists {
		r.Store[key] = strconv.Itoa(increment) // Create the key with the increment if not exists
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
func (r *RedisClone) GetRange(key string, start, end int) string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Get the value of the key
	value, exists := r.Store[key]
	if !exists {
		return ""
	}

	// Handle negative indices
	if start < 0 {
		start = len(value) + start
	}
	if end < 0 {
		end = len(value) + end
	}

	// Ensure start and end are within the bounds of the string
	if start < 0 {
		start = 0
	}
	if start >= len(value) {
		return ""
	}

	if end >= len(value) {
		end = len(value) - 1
	}

	// Return the substring
	if start > end {
		return ""
	}

	return value[start : end+1]
}

func (r *RedisClone) SetRange(key string, offset int, value string) int {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Get the current value for the key, or create an empty string if not found
	currentValue, exists := r.Store[key]
	if !exists {
		currentValue = ""
	}

	// If the offset is larger than the current length, pad the string with null bytes
	if offset > len(currentValue) {
		currentValue = currentValue + strings.Repeat(" ", offset-len(currentValue)) // Add spaces only up to the offset
	}

	// Update the string starting from the offset
	newValue := currentValue[:offset] + value

	// Set the new value in the store
	r.Store[key] = newValue

	// Return the length of the new string
	return len(newValue)
}

func (r *RedisClone) Keys(pattern string) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var matchedKeys []string

	for key := range r.Store {
		// Match all keys if pattern is "*"
		if pattern == "*" || matchesPattern(key, pattern) {
			matchedKeys = append(matchedKeys, key)
		}
	}
	return matchedKeys
}

// Helper function to match a key against a pattern
func matchesPattern(key, pattern string) bool {
	// Use path.Match for glob-style pattern matching
	matched, err := path.Match(pattern, key)
	return err == nil && matched
}

func (r *RedisClone) JSONSet(key, path string, value interface{}) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Parse JSON if the key exists, or create a new map
	var jsonData map[string]interface{}
	if existing, ok := r.Store[key]; ok {
		if err := json.Unmarshal([]byte(existing), &jsonData); err != nil {
			return fmt.Errorf("invalid JSON data at key %s", key)
		}
	} else {
		jsonData = make(map[string]interface{})
	}

	// Set the value at the specified path
	if err := setJSONValue(jsonData, path, value); err != nil {
		return err
	}

	// Serialize back to JSON and store
	serialized, err := json.Marshal(jsonData)
	if err != nil {
		return fmt.Errorf("failed to serialize JSON: %v", err)
	}
	r.Store[key] = string(serialized)
	return nil
}

func setJSONValue(data interface{}, path string, value interface{}) error {
	if !strings.HasPrefix(path, "$.") {
		return fmt.Errorf("invalid path: must start with $.")
	}

	segments := strings.Split(path[2:], ".")
	current := data

	for i, segment := range segments {
		if i == len(segments)-1 { // Last segment
			switch node := current.(type) {
			case map[string]interface{}:
				node[segment] = value
				return nil
			default:
				return fmt.Errorf("cannot set value at path: %s", path)
			}
		}

		switch node := current.(type) {
		case map[string]interface{}:
			next, exists := node[segment]
			if !exists {
				// Create a new nested object if it doesn't exist
				newMap := make(map[string]interface{})
				node[segment] = newMap
				current = newMap
			} else {
				current = next
			}
		default:
			return fmt.Errorf("invalid structure at path segment: %s", segment)
		}
	}

	return nil
}

func getJSONValue(data interface{}, path string) (interface{}, error) {
	if !strings.HasPrefix(path, "$.") {
		return nil, fmt.Errorf("invalid path: must start with $.")
	}

	segments := strings.Split(path[2:], ".")
	current := data

	for _, segment := range segments {
		switch node := current.(type) {
		case map[string]interface{}:
			next, exists := node[segment]
			if !exists {
				return nil, fmt.Errorf("path not found: %s", path)
			}
			current = next
		case []interface{}:
			index, err := strconv.Atoi(segment)
			if err != nil || index < 0 || index >= len(node) {
				return nil, fmt.Errorf("invalid array index at path segment: %s", segment)
			}
			current = node[index]
		default:
			return nil, fmt.Errorf("invalid structure at path segment: %s", segment)
		}
	}

	return current, nil
}
func deleteJSONValue(data interface{}, path string) error {
	if !strings.HasPrefix(path, "$.") {
		return fmt.Errorf("invalid path: must start with $.")
	}

	segments := strings.Split(path[2:], ".")
	current := data

	for i, segment := range segments {
		if i == len(segments)-1 { // Last segment
			switch node := current.(type) {
			case map[string]interface{}:
				delete(node, segment)
				return nil
			default:
				return fmt.Errorf("cannot delete value at path: %s", path)
			}
		}

		switch node := current.(type) {
		case map[string]interface{}:
			next, exists := node[segment]
			if !exists {
				return fmt.Errorf("path not found: %s", path)
			}
			current = next
		default:
			return fmt.Errorf("invalid structure at path segment: %s", segment)
		}
	}

	return nil
}
func appendJSONArray(data interface{}, path string, values []interface{}) error {
	target, err := getJSONValue(data, path)
	if err != nil {
		return err
	}

	array, ok := target.([]interface{})
	if !ok {
		return fmt.Errorf("target at path %s is not an array", path)
	}

	array = append(array, values...)
	// Set the updated array back
	return setJSONValue(data, path, array)
}
func (r *RedisClone) JSONGet(key, path string) (interface{}, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	existing, ok := r.Store[key]
	if !ok {
		return nil, fmt.Errorf("key %s not found", key)
	}

	var jsonData interface{}
	if err := json.Unmarshal([]byte(existing), &jsonData); err != nil {
		return nil, fmt.Errorf("invalid JSON data at key %s", key)
	}

	// Get value at the specified path
	return getJSONValue(jsonData, path)
}
func (r *RedisClone) JSONDel(key, path string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	existing, ok := r.Store[key]
	if !ok {
		return fmt.Errorf("key %s not found", key)
	}

	var jsonData map[string]interface{}
	if err := json.Unmarshal([]byte(existing), &jsonData); err != nil {
		return fmt.Errorf("invalid JSON data at key %s", key)
	}

	// Delete the value at the specified path
	if err := deleteJSONValue(jsonData, path); err != nil {
		return err
	}

	// Serialize back to JSON and update
	serialized, err := json.Marshal(jsonData)
	if err != nil {
		return fmt.Errorf("failed to serialize JSON: %v", err)
	}
	r.Store[key] = string(serialized)
	return nil
}
func (r *RedisClone) JSONArrAppend(key, path string, values ...interface{}) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	existing, ok := r.Store[key]
	if !ok {
		return fmt.Errorf("key %s not found", key)
	}

	var jsonData interface{}
	if err := json.Unmarshal([]byte(existing), &jsonData); err != nil {
		return fmt.Errorf("invalid JSON data at key %s", key)
	}

	// Append to array at the specified path
	if err := appendJSONArray(jsonData, path, values); err != nil {
		return err
	}

	// Serialize back to JSON and update
	serialized, err := json.Marshal(jsonData)
	if err != nil {
		return fmt.Errorf("failed to serialize JSON: %v", err)
	}
	r.Store[key] = string(serialized)
	return nil
}
