package storage

import (
	"fmt"
	"sync"
	"time"
)

type StreamEntry struct {
	ID     string
	Fields map[string]string
}

type Stream struct {
	Entries        []StreamEntry
	ConsumerGroups map[string]*ConsumerGroup
	mu             sync.RWMutex
}

type ConsumerGroup struct {
	Consumers map[string]*Consumer
	Pending   map[string]StreamEntry
}

type Consumer struct {
	Pending []string
}

// --- Stream Operations ---

// XAdd adds an entry to the stream.
func (r *RedisClone) XAdd(key string, id string, fields map[string]string) string {
	r.Mu.Lock()
	defer r.Mu.Unlock()

	stream, ok := r.Store[key].(*Stream)
	if !ok {
		stream = &Stream{
			Entries:        []StreamEntry{},
			ConsumerGroups: make(map[string]*ConsumerGroup),
		}
		r.Store[key] = stream
	}

	if id == "*" {
		id = fmt.Sprintf("%d-0", time.Now().UnixNano())
	}

	entry := StreamEntry{ID: id, Fields: fields}
	stream.mu.Lock()
	stream.Entries = append(stream.Entries, entry)
	stream.mu.Unlock()

	return id
}

// XRead reads entries from streams.
func (r *RedisClone) XRead(key string, startID string, count int) []StreamEntry {
	r.Mu.RLock()
	defer r.Mu.RUnlock()

	stream, ok := r.Store[key].(*Stream)
	if !ok {
		return nil
	}

	stream.mu.RLock()
	defer stream.mu.RUnlock()

	var result []StreamEntry
	for _, entry := range stream.Entries {
		entryId := entry.ID
		print(entryId)
		if entry.ID > startID {
			result = append(result, entry)
			if count > 0 && len(result) >= count {
				break
			}
		}
	}
	return result
}

// XRange retrieves entries within a range.
func (r *RedisClone) XRange(key, startID, endID string) []StreamEntry {
	r.Mu.RLock()
	defer r.Mu.RUnlock()

	stream, ok := r.Store[key].(*Stream)
	if !ok {
		return nil
	}

	stream.mu.RLock()
	defer stream.mu.RUnlock()

	var result []StreamEntry
	for _, entry := range stream.Entries {
		if entry.ID >= startID && entry.ID <= endID {
			result = append(result, entry)
		}
	}
	return result
}

// XLen returns the length of the stream.
func (r *RedisClone) XLen(key string) int {
	r.Mu.RLock()
	defer r.Mu.RUnlock()

	stream, ok := r.Store[key].(*Stream)
	if !ok {
		return 0
	}

	stream.mu.RLock()
	defer stream.mu.RUnlock()

	return len(stream.Entries)
}

// --- Consumer Group Operations ---

// XGroupCreate CREATE creates a consumer group.
func (r *RedisClone) XGroupCreate(key, groupName string) bool {
	r.Mu.Lock()
	defer r.Mu.Unlock()

	stream, ok := r.Store[key].(*Stream)
	if !ok {
		return false
	}

	stream.mu.Lock()
	defer stream.mu.Unlock()

	if _, exists := stream.ConsumerGroups[groupName]; exists {
		return false
	}

	stream.ConsumerGroups[groupName] = &ConsumerGroup{
		Consumers: make(map[string]*Consumer),
		Pending:   make(map[string]StreamEntry),
	}
	return true
}

// XReadGroup reads entries for a consumer in a group.
func (r *RedisClone) XReadGroup(key, groupName, consumerName, startID string, count int) []StreamEntry {
	r.Mu.Lock()
	defer r.Mu.Unlock()

	stream, ok := r.Store[key].(*Stream)
	if !ok {
		return nil
	}

	stream.mu.Lock()
	defer stream.mu.Unlock()

	group, exists := stream.ConsumerGroups[groupName]
	if !exists {
		return nil
	}

	consumer, exists := group.Consumers[consumerName]
	if !exists {
		consumer = &Consumer{Pending: []string{}}
		group.Consumers[consumerName] = consumer
	}

	var result []StreamEntry
	for _, entry := range stream.Entries {
		if entry.ID > startID {
			result = append(result, entry)
			group.Pending[entry.ID] = entry
			consumer.Pending = append(consumer.Pending, entry.ID)
			if count > 0 && len(result) >= count {
				break
			}
		}
	}
	return result
}

// XAck acknowledges messages for a consumer group.
func (r *RedisClone) XAck(key, groupName string, ids []string) int {
	r.Mu.Lock()
	defer r.Mu.Unlock()

	stream, ok := r.Store[key].(*Stream)
	if !ok {
		return 0
	}

	stream.mu.Lock()
	defer stream.mu.Unlock()

	group, exists := stream.ConsumerGroups[groupName]
	if !exists {
		return 0
	}

	ackCount := 0
	for _, id := range ids {
		if _, exists := group.Pending[id]; exists {
			delete(group.Pending, id)
			ackCount++
		}
	}
	return ackCount
}
