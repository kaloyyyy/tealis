package storage

import (
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

func (r *RedisClone) StartCleanup() {
	go func() {
		for {
			time.Sleep(time.Second) // Run cleanup every second
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
	}()
}
