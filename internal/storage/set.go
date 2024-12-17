package storage

// SADD adds one or more members to a set.
func (r *RedisClone) SADD(key string, members ...string) int {
	r.Mu.Lock()
	defer r.Mu.Unlock()

	// Initialize the set if not already created
	if _, exists := r.Store[key]; !exists {
		r.Store[key] = make(map[string]struct{})
	}

	// Assert the value as a map of strings (set)
	set := r.Store[key].(map[string]struct{})

	// Add members to the set
	for _, member := range members {
		set[member] = struct{}{}
	}

	return len(set)
}

// SREM removes one or more members from a set.
func (r *RedisClone) SREM(key string, members ...string) int {
	r.Mu.Lock()
	defer r.Mu.Unlock()

	set, exists := r.Store[key].(map[string]struct{})
	if !exists {
		return 0
	}

	// Remove members from the set
	count := 0
	for _, member := range members {
		if _, exists := set[member]; exists {
			delete(set, member)
			count++
		}
	}

	return count
}

// SISMEMBER checks if a member exists in the set.
func (r *RedisClone) SISMEMBER(key, member string) bool {
	r.Mu.RLock()
	defer r.Mu.RUnlock()

	set, exists := r.Store[key].(map[string]struct{})
	if !exists {
		return false
	}

	_, exists = set[member]
	return exists
}

// SMEMBERS returns all members of a set.
func (r *RedisClone) SMEMBERS(key string) []string {
	r.Mu.RLock()
	defer r.Mu.RUnlock()

	set, exists := r.Store[key].(map[string]struct{})
	if !exists {
		return nil
	}

	// Convert the set to a slice
	var members []string
	for member := range set {
		members = append(members, member)
	}

	return members
}

// SUNION returns the union of multiple sets.
func (r *RedisClone) SUNION(keys ...string) []string {
	r.Mu.RLock()
	defer r.Mu.RUnlock()

	union := make(map[string]struct{})
	for _, key := range keys {
		set, exists := r.Store[key].(map[string]struct{})
		if !exists {
			continue
		}

		// Add all members of the set to the union
		for member := range set {
			union[member] = struct{}{}
		}
	}

	// Convert the union map to a slice
	var members []string
	for member := range union {
		members = append(members, member)
	}

	return members
}

// SINTER returns the intersection of multiple sets.
func (r *RedisClone) SINTER(keys ...string) []string {
	r.Mu.RLock()
	defer r.Mu.RUnlock()

	if len(keys) == 0 {
		return nil
	}

	// Get the first set
	firstSet, exists := r.Store[keys[0]].(map[string]struct{})
	if !exists {
		return nil
	}

	// Intersect the first set with the others
	intersection := make(map[string]struct{})
	for member := range firstSet {
		intersection[member] = struct{}{}
	}

	// For each subsequent set, keep only the members that are common
	for _, key := range keys[1:] {
		set, exists := r.Store[key].(map[string]struct{})
		if !exists {
			return nil
		}

		for member := range intersection {
			if _, exists := set[member]; !exists {
				delete(intersection, member)
			}
		}
	}

	// Convert the intersection map to a slice
	var members []string
	for member := range intersection {
		members = append(members, member)
	}

	return members
}

// SDIFF returns the difference between multiple sets.
func (r *RedisClone) SDIFF(keys ...string) []string {
	r.Mu.RLock()
	defer r.Mu.RUnlock()

	if len(keys) == 0 {
		return nil
	}

	// Get the first set
	firstSet, exists := r.Store[keys[0]].(map[string]struct{})
	if !exists {
		return nil
	}

	// Store the difference
	difference := make(map[string]struct{})
	for member := range firstSet {
		difference[member] = struct{}{}
	}

	// Subtract the other sets
	for _, key := range keys[1:] {
		set, exists := r.Store[key].(map[string]struct{})
		if !exists {
			continue
		}

		for member := range set {
			delete(difference, member)
		}
	}

	// Convert the difference map to a slice
	var members []string
	for member := range difference {
		members = append(members, member)
	}

	return members
}
