package storage

import (
	"fmt"
	"path"
	"strconv"
	"strings"
	"time"
)

// Set saves a key-value pair with an optional TTL.
func (r *RedisClone) Set(key, value string, ttl time.Duration) {
	r.Mu.Lock()
	defer r.Mu.Unlock()

	r.Store[key] = value
	if ttl > 0 {
		r.Expiries[key] = time.Now().Add(ttl)
	} else {
		delete(r.Expiries, key)
	}
}

// Get retrieves the value for a key.
func (r *RedisClone) Get(key string) (string, bool) {
	r.Mu.RLock()
	defer r.Mu.RUnlock()

	if expiry, exists := r.Expiries[key]; exists && time.Now().After(expiry) {
		r.Mu.Lock()
		delete(r.Store, key)
		delete(r.Expiries, key)
		r.Mu.Unlock()
		return "", false
	}

	value, exists := r.Store[key]
	if value != nil {
		return value.(string), exists
	}
	return "", exists
}

// Del deletes a key from the store.
func (r *RedisClone) Del(key string) bool {
	r.Mu.Lock()
	defer r.Mu.Unlock()

	if _, exists := r.Store[key]; exists {
		delete(r.Store, key)
		delete(r.Expiries, key)
		return true
	}
	return false
}

// Exists checks if a key exists in the store.
func (r *RedisClone) Exists(key string) bool {
	r.Mu.RLock()
	defer r.Mu.RUnlock()
	_, exists := r.Store[key]
	return exists
}

// Append appends a value to an existing key.
func (r *RedisClone) Append(key, value string) int {
	r.Mu.Lock()
	defer r.Mu.Unlock()

	if current, exists := r.Store[key]; exists {
		r.Store[key] = current.(string) + value
	} else {
		r.Store[key] = value
	}
	return len(r.Store[key].(string))
}

// StrLen returns the length of a string value for a key.
func (r *RedisClone) StrLen(key string) int {
	r.Mu.RLock()
	defer r.Mu.RUnlock()

	if value, exists := r.Store[key]; exists {
		return len(value.(string))
	}
	return 0
}

// IncrBy increments a key by a specified value.
func (r *RedisClone) IncrBy(key string, increment int) (int, error) {
	r.Mu.Lock()
	defer r.Mu.Unlock()

	current, exists := r.Store[key]
	if !exists {
		r.Store[key] = strconv.Itoa(increment)
		return increment, nil
	}

	currentInt, err := strconv.Atoi(current.(string))
	if err != nil {
		return 0, fmt.Errorf("value is not an integer")
	}

	newValue := currentInt + increment
	r.Store[key] = strconv.Itoa(newValue)
	return newValue, nil
}

// GetRange retrieves a substring from a value.
func (r *RedisClone) GetRange(key string, start, end int) string {
	r.Mu.RLock()
	defer r.Mu.RUnlock()

	value, exists := r.Store[key].(string)
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
	r.Mu.Lock()
	defer r.Mu.Unlock()

	currentValue, exists := r.Store[key].(string)
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
	r.Mu.RLock()
	defer r.Mu.RUnlock()

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
