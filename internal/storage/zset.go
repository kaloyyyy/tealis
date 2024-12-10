package storage

import "sort"

// SortedSetEntry represents an entry in the sorted set.
type SortedSetEntry struct {
	Score  float64
	Member string
}

// SortedSet is a sorted set structure.
type SortedSet struct {
	members map[string]float64 // Map of members to their scores
	sorted  []SortedSetEntry   // Sorted list of entries for range queries
}

// ZADD adds a member with a score to a sorted set.
func (r *RedisClone) ZADD(key string, score float64, member string) int {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Retrieve or create the sorted set
	var set *SortedSet
	if rawSet, exists := r.Store[key]; exists {
		set = rawSet.(*SortedSet)
	} else {
		set = &SortedSet{members: make(map[string]float64)}
		r.Store[key] = set
	}

	// Add or update the member
	_, exists := set.members[member]
	set.members[member] = score
	set.updateSorted()

	if exists {
		return 0 // Updated
	}
	return 1 // New addition
}

// ZRANGE returns a range of members by rank.
func (r *RedisClone) ZRANGE(key string, start, stop int) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	rawSet, exists := r.Store[key]
	if !exists {
		return nil
	}
	set := rawSet.(*SortedSet)

	// Adjust for negative indices
	if start < 0 {
		start += len(set.sorted)
	}
	if stop < 0 {
		stop += len(set.sorted)
	}
	if start < 0 {
		start = 0
	}
	if stop >= len(set.sorted) {
		stop = len(set.sorted) - 1
	}

	// Extract members in the range
	if start > stop {
		return nil
	}
	members := []string{}
	for _, entry := range set.sorted[start : stop+1] {
		members = append(members, entry.Member)
	}
	return members
}

// ZRANK returns the rank of a member in the sorted set.
func (r *RedisClone) ZRANK(key, member string) int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	rawSet, exists := r.Store[key]
	if !exists {
		return -1
	}
	set := rawSet.(*SortedSet)

	for i, entry := range set.sorted {
		if entry.Member == member {
			return i
		}
	}
	return -1
}

// ZREM removes a member from the sorted set.
func (r *RedisClone) ZREM(key, member string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	rawSet, exists := r.Store[key]
	if !exists {
		return false
	}
	set := rawSet.(*SortedSet)

	_, exists = set.members[member]
	if !exists {
		return false
	}

	delete(set.members, member)
	set.updateSorted()
	return true
}

// ZRANGEBYSCORE returns members with scores within a specified range.
func (r *RedisClone) ZRANGEBYSCORE(key string, min, max float64) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	rawSet, exists := r.Store[key]
	if !exists {
		return nil
	}
	set := rawSet.(*SortedSet)

	members := []string{}
	for _, entry := range set.sorted {
		if entry.Score >= min && entry.Score <= max {
			members = append(members, entry.Member)
		}
	}
	return members
}

func (s *SortedSet) updateSorted() {
	s.sorted = s.sorted[:0]
	for member, score := range s.members {
		s.sorted = append(s.sorted, SortedSetEntry{Score: score, Member: member})
	}
	sort.Slice(s.sorted, func(i, j int) bool {
		return s.sorted[i].Score < s.sorted[j].Score
	})
}
