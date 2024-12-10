package storage

func (r *RedisClone) HSET(key, field string, value interface{}) int {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Retrieve or create the hash
	hash, ok := r.Store[key].(map[string]interface{})
	if !ok {
		hash = make(map[string]interface{})
		r.Store[key] = hash
	}

	// Check if the field already exists
	_, exists := hash[field]

	// Set the field to the new value
	hash[field] = value

	// Return 1 if a new field was added, 0 if the field was updated
	if exists {
		return 0
	}
	return 1
}
func (r *RedisClone) HGET(key, field string) (interface{}, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Retrieve the hash
	hash, ok := r.Store[key].(map[string]interface{})
	if !ok {
		return nil, false
	}

	// Retrieve the field's value
	value, exists := hash[field]
	return value, exists
}

func (r *RedisClone) HMSET(key string, fields map[string]interface{}) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Retrieve or create the hash
	hash, ok := r.Store[key].(map[string]interface{})
	if !ok {
		hash = make(map[string]interface{})
		r.Store[key] = hash
	}

	// Set all fields in the hash
	for field, value := range fields {
		hash[field] = value
	}
}
func (r *RedisClone) HGETALL(key string) map[string]interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Retrieve the hash
	hash, ok := r.Store[key].(map[string]interface{})
	if !ok {
		return nil
	}

	// Return a copy of the hash
	result := make(map[string]interface{})
	for field, value := range hash {
		result[field] = value
	}
	return result
}

func (r *RedisClone) HDEL(key string, field string) int {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Retrieve the hash
	hash, ok := r.Store[key].(map[string]interface{})
	if !ok {
		return 0
	}

	// Delete the field
	if _, exists := hash[field]; exists {
		delete(hash, field)
		return 1
	}
	return 0
}

func (r *RedisClone) HEXISTS(key, field string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Retrieve the hash
	hash, ok := r.Store[key].(map[string]interface{})
	if !ok {
		return false
	}

	// Check if the field exists
	_, exists := hash[field]
	return exists
}
