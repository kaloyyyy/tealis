package storage

import (
	_ "fmt"
	_ "sync"
	_ "time"
)

// RPUSH appends one or more values to the end of a list.
func (r *RedisClone) RPUSH(key string, values ...string) int {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Initialize the list if not already created
	if _, exists := r.Store[key]; !exists {
		r.Store[key] = []string{}
	}

	// Assert the value as a slice of strings
	list := r.Store[key].([]string)
	r.Store[key] = append(list, values...)
	return len(r.Store[key].([]string))
}

// LPUSH prepends one or more values to the beginning of a list.
func (r *RedisClone) LPUSH(key string, values ...string) int {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Initialize the list if not already created
	if _, exists := r.Store[key]; !exists {
		r.Store[key] = []string{}
	}

	// Assert the value as a slice of strings
	list := r.Store[key].([]string)
	r.Store[key] = append(values, list...)
	return len(r.Store[key].([]string))
}

// LPOP removes and returns the first element of the list.
func (r *RedisClone) LPOP(key string) (string, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	list, exists := r.Store[key].([]string)
	if !exists || len(list) == 0 {
		return "", false
	}

	// Pop the first element
	r.Store[key] = list[1:]
	return list[0], true
}

// RPOP removes and returns the last element of the list.
func (r *RedisClone) RPOP(key string) (string, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	list, exists := r.Store[key].([]string)
	if !exists || len(list) == 0 {
		return "", false
	}

	// Pop the last element
	r.Store[key] = list[:len(list)-1]
	return list[len(list)-1], true
}

// LRANGE returns a slice of elements in the list within the specified range.
func (r *RedisClone) LRANGE(key string, start, stop int) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	list, exists := r.Store[key].([]string)
	if !exists {
		return nil
	}

	// Handle negative indexing
	if start < 0 {
		start = len(list) + start
	}
	if stop < 0 {
		stop = len(list) + stop
	}

	// Handle the range boundaries
	if start < 0 {
		start = 0
	}
	if stop >= len(list) {
		stop = len(list) - 1
	}

	if start > stop {
		return nil
	}

	return list[start : stop+1]
}
