package storage

import (
	"context"
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
