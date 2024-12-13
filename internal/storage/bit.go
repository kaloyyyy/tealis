package storage

import (
	"math/bits"
)

// SETBIT sets the bit at the specified offset in the key's value.
func (r *RedisClone) SETBIT(key string, offset, value int) int {
	r.mu.Lock()
	defer r.mu.Unlock()

	if offset < 0 {
		return 0 // Invalid offset
	}

	// Ensure the value is a byte slice
	data, _ := r.Store[key].([]byte)
	byteIndex := offset / 8
	bitIndex := offset % 8

	// Expand the byte slice if necessary
	if len(data) <= byteIndex {
		newData := make([]byte, byteIndex+1)
		copy(newData, data)
		data = newData
	}

	// Get the previous bit value
	mask := byte(1 << bitIndex)
	prev := (data[byteIndex] & mask) >> bitIndex

	// Set or clear the bit
	if value == 1 {
		data[byteIndex] |= mask
	} else {
		data[byteIndex] &^= mask
	}

	// Update the store
	r.Store[key] = data
	return int(prev)
}

// GETBIT retrieves the bit at the specified offset in the key's value.
func (r *RedisClone) GETBIT(key string, offset int) int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if offset < 0 {
		return 0
	}

	data, _ := r.Store[key].([]byte)
	byteIndex := offset / 8
	if byteIndex >= len(data) {
		return 0 // Out of range, default to 0
	}

	bitIndex := offset % 8
	return int((data[byteIndex] >> bitIndex) & 1)
}

// BITCOUNT counts the number of bits set to 1 in the key's value.
func (r *RedisClone) BITCOUNT(key string) int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	data, _ := r.Store[key].([]byte)
	count := 0
	for _, b := range data {
		count += bits.OnesCount8(b)
	}
	return count
}

// BITOP performs bitwise operations between keys and stores the result in a destination key.
func (r *RedisClone) BITOP(op string, destKey string, keys ...string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	var result []byte
	for i, key := range keys {
		data, _ := r.Store[key].([]byte)

		// Align the result size with the current key's data size
		if len(result) < len(data) {
			newResult := make([]byte, len(data))
			copy(newResult, result)
			result = newResult
		} else if len(result) > len(data) {
			newData := make([]byte, len(result))
			copy(newData, data)
			data = newData
		}

		// Perform the operation
		for j := range result {
			switch op {
			case "AND":
				result[j] &= data[j]
			case "OR":
				result[j] |= data[j]
			case "XOR":
				result[j] ^= data[j]
			case "NOT":
				if i != 0 {
					panic("-ERR NOT must be applied to a single key")
				}
				result[j] = ^data[j]
			}
		}
	}

	// Store the result in the destination key
	r.Store[destKey] = result
}
