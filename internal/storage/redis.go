package storage

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
)

type RedisClone struct {
	mu       sync.RWMutex
	store    map[string]string
	expiries map[string]time.Time
}

func NewRedisClone() *RedisClone {
	return &RedisClone{
		store:    make(map[string]string),
		expiries: make(map[string]time.Time),
	}
}

func (r *RedisClone) Set(key, value string, ttl time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.store[key] = value
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
		delete(r.store, key)
		delete(r.expiries, key)
		r.mu.Unlock()
		return "", false
	}

	value, exists := r.store[key]
	return value, exists
}

func (r *RedisClone) Del(key string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.store[key]; exists {
		delete(r.store, key)
		delete(r.expiries, key)
		return true
	}
	return false
}

// Exists checks if a key exists in the store.
func (r *RedisClone) Exists(key string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, exists := r.store[key]
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
						delete(r.store, key)
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

	if current, exists := r.store[key]; exists {
		r.store[key] = current + value
	} else {
		r.store[key] = value
	}
	return len(r.store[key])
}

func (r *RedisClone) StrLen(key string) int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if value, exists := r.store[key]; exists {
		return len(value)
	}
	return 0
}
func (r *RedisClone) IncrBy(key string, increment int) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	current, exists := r.store[key]
	if !exists {
		r.store[key] = strconv.Itoa(increment) // Create the key with the increment if not exists
		return increment, nil
	}

	currentInt, err := strconv.Atoi(current)
	if err != nil {
		return 0, fmt.Errorf("value is not an integer")
	}

	newValue := currentInt + increment
	r.store[key] = strconv.Itoa(newValue)
	return newValue, nil
}
func (r *RedisClone) GetRange(key string, start, end int) string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Get the value of the key
	value, exists := r.store[key]
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
	currentValue, exists := r.store[key]
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
	r.store[key] = newValue

	// Return the length of the new string
	return len(newValue)
}
