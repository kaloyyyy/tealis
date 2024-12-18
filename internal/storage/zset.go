package storage

import (
	"math/rand"
	"sync"
	"time"
)

const maxLevel = 16 // Maximum levels for the skip list

// SortedSet represents a sorted set with a skip list.
type SortedSet struct {
	mu      sync.RWMutex
	header  *skipListNode
	level   int
	length  int
	randSrc rand.Source
}

// skipListNode represents a single node in the skip list.
type skipListNode struct {
	key     string
	score   float64
	forward []*skipListNode
}

// NewSortedSet initializes a new sorted set.
func NewSortedSet() *SortedSet {
	return &SortedSet{
		header:  newSkipListNode("", 0, maxLevel),
		level:   1,
		randSrc: rand.NewSource(time.Now().UnixNano()),
	}
}

func newSkipListNode(key string, score float64, level int) *skipListNode {
	return &skipListNode{
		key:     key,
		score:   score,
		forward: make([]*skipListNode, level),
	}
}

// randomLevel generates a random level for the node.
func (s *SortedSet) randomLevel() int {
	level := 1
	for rand.New(s.randSrc).Float32() < 0.5 && level < maxLevel {
		level++
	}
	return level
}

func (s *SortedSet) ZAdd(key string, score float64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	update := make([]*skipListNode, maxLevel)
	current := s.header

	// Find the position to insert the new node
	for i := s.level - 1; i >= 0; i-- {
		for current.forward[i] != nil && (current.forward[i].score < score || (current.forward[i].score == score && current.forward[i].key < key)) {
			current = current.forward[i]
		}
		update[i] = current
	}

	// Check if the key already exists
	if current.forward[0] != nil && current.forward[0].key == key {
		current.forward[0].score = score // Update the score if the key exists
		return
	}

	// Insert new node
	newLevel := s.randomLevel()
	if newLevel > s.level {
		for i := s.level; i < newLevel; i++ {
			update[i] = s.header
		}
		s.level = newLevel
	}
	node := newSkipListNode(key, score, newLevel)
	for i := 0; i < newLevel; i++ {
		node.forward[i] = update[i].forward[i]
		update[i].forward[i] = node
	}
	s.length++
}

func (s *SortedSet) ZRange(start, end int) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if start < 0 || end < 0 || start >= s.length || start > end {
		return nil
	}

	result := []string{}
	current := s.header.forward[0]
	for i := 0; i < start && current != nil; i++ {
		current = current.forward[0]
	}

	for i := start; i <= end && current != nil; i++ {
		result = append(result, current.key)
		current = current.forward[0]
	}
	return result
}

func (s *SortedSet) ZRank(key string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	current := s.header
	rank := 0

	for i := s.level - 1; i >= 0; i-- {
		for current.forward[i] != nil && current.forward[i].key < key {
			rank++
			current = current.forward[i]
		}
	}
	if current.forward[0] != nil && current.forward[0].key == key {
		return rank
	}
	return -1 // Key not found
}

func (s *SortedSet) ZRem(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	update := make([]*skipListNode, maxLevel)
	current := s.header
	for i := s.level - 1; i >= 0; i-- {
		for current.forward[i] != nil && current.forward[i].key < key {
			current = current.forward[i]
		}
		update[i] = current
	}

	target := current.forward[0]
	if target == nil || target.key != key {
		return false // Key not found
	}

	for i := 0; i < s.level; i++ {
		if update[i].forward[i] != target {
			break
		}
		update[i].forward[i] = target.forward[i]
	}

	// Adjust the level of the skip list
	for s.level > 1 && s.header.forward[s.level-1] == nil {
		s.level--
	}
	s.length--
	return true
}

func (s *SortedSet) ZRangeByScore(min, max float64) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := []string{}
	current := s.header.forward[0]

	for current != nil && current.score < min {
		current = current.forward[0]
	}

	for current != nil && current.score <= max {
		result = append(result, current.key)
		current = current.forward[0]
	}
	return result
}

func (r *RedisClone) ZAdd(key string, score float64, member string) int {
	r.Mu.Lock()
	defer r.Mu.Unlock()

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
	r.Mu.RLock()
	defer r.Mu.RUnlock()

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
	r.Mu.RLock()
	defer r.Mu.RUnlock()

	if val, exists := r.Store[key]; exists {
		if ss, ok := val.(*SortedSet); ok {
			return ss.ZRank(member)
		}
		panic("WRONGTYPE Operation against a key holding the wrong kind of value")
	}
	return -1 // Key does not exist
}

func (r *RedisClone) ZRem(key string, member string) bool {
	r.Mu.Lock()
	defer r.Mu.Unlock()

	if val, exists := r.Store[key]; exists {
		if ss, ok := val.(*SortedSet); ok {
			return ss.ZRem(member)
		}
		panic("WRONGTYPE Operation against a key holding the wrong kind of value")
	}
	return false // Key does not exist
}

func (r *RedisClone) ZRangeByScore(key string, min, max float64) []string {
	r.Mu.RLock()
	defer r.Mu.RUnlock()

	if val, exists := r.Store[key]; exists {
		if ss, ok := val.(*SortedSet); ok {
			return ss.ZRangeByScore(min, max)
		}
		panic("WRONGTYPE Operation against a key holding the wrong kind of value")
	}
	return nil // Key does not exist
}
