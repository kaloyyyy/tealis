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
