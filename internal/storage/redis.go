package storage

import (
	"context"
	"sync"
	"time"
)

// RedisClone represents the basic structure for a Redis-like store.
type RedisClone struct {
	mu       sync.RWMutex
	Store    map[string]interface{} // Store can hold any data type (string, list, etc.)
	expiries map[string]time.Time
}

// NewRedisClone initializes a new RedisClone instance.
func NewRedisClone() *RedisClone {
	return &RedisClone{
		Store:    make(map[string]interface{}),
		expiries: make(map[string]time.Time),
	}
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

func (r *RedisClone) ZAdd(key string, score float64, member string) int {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if the key exists and is a sorted set
	if val, exists := r.Store[key]; exists {
		if ss, ok := val.(*SortedSet); ok {
			ss.ZAdd(member, score)
			return 1
		}
		panic("WRONGTYPE Operation against a key holding the wrong kind of value")
	}

	// If key doesn't exist, create a new SortedSet
	ss := NewSortedSet()
	ss.ZAdd(member, score)
	r.Store[key] = ss
	return 1
}

func (r *RedisClone) ZRange(key string, start, end int) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Check if the key exists and is a sorted set
	if val, exists := r.Store[key]; exists {
		if ss, ok := val.(*SortedSet); ok {
			return ss.ZRange(start, end)
		}
		panic("WRONGTYPE Operation against a key holding the wrong kind of value")
	}
	return nil // Key does not exist
}

func (r *RedisClone) ZRank(key string, member string) int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if val, exists := r.Store[key]; exists {
		if ss, ok := val.(*SortedSet); ok {
			return ss.ZRank(member)
		}
		panic("WRONGTYPE Operation against a key holding the wrong kind of value")
	}
	return -1 // Key does not exist
}

func (r *RedisClone) ZRem(key string, member string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	if val, exists := r.Store[key]; exists {
		if ss, ok := val.(*SortedSet); ok {
			return ss.ZRem(member)
		}
		panic("WRONGTYPE Operation against a key holding the wrong kind of value")
	}
	return false // Key does not exist
}

func (r *RedisClone) ZRangeByScore(key string, min, max float64) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if val, exists := r.Store[key]; exists {
		if ss, ok := val.(*SortedSet); ok {
			return ss.ZRangeByScore(min, max)
		}
		panic("WRONGTYPE Operation against a key holding the wrong kind of value")
	}
	return nil // Key does not exist
}
