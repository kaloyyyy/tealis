package storage

import (
	"fmt"
	"strings"
)

// BITFIELD handles bitfield operations for GET, SET, INCRBY
func (r *RedisClone) BITFIELD(operation string, key string, bitType string, list []int) ([]interface{}, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	op := strings.ToUpper(operation)

	// Fetch the existing bitfield data for the given key
	data, ok := r.Store[key].([]byte)
	if !ok {
		data = make([]byte, 0)
		r.Store[key] = data
	}

	var result []interface{}

	// Ensure the list has enough elements
	if len(list) < 1 {
		return nil, fmt.Errorf("-ERR invalid number of arguments for BITFIELD operation")
	}

	// Parse the operation type, bitType, and offset
	offset := list[0]

	var size int
	switch bitType {
	case "i32":
		size = 4 // 32-bit integer (4 bytes)
	case "i64":
		size = 8 // 64-bit integer (8 bytes)
	default:
		return nil, fmt.Errorf("-ERR invalid bit type %s", bitType)
	}

	switch op {
	case "GET":
		// Retrieve the value of the counter at the specified offset
		if offset+size > len(data)*8 {
			return nil, fmt.Errorf("-ERR offset out of range")
		}

		// Extract the value from the bitfield
		value := int64(0)
		for j := 0; j < size; j++ {
			// Read each bit starting from the offset
			bit := (data[offset/8] >> (7 - offset%8)) & 1
			value |= int64(bit) << (size - 1 - j)
			offset++
		}

		result = append(result, value)

	case "SET":
		// Set the value at the specified offset
		if len(list) < 2 {
			return nil, fmt.Errorf("-ERR invalid number of arguments for BITFIELD SET operation")
		}
		value := list[1]

		// Resize the bitfield if necessary
		if offset+size > len(data)*8 {
			newSize := (offset + size + 7) / 8
			newData := make([]byte, newSize)
			copy(newData, data)
			data = newData
			r.Store[key] = data
		}

		// Set the value at the offset
		for j := 0; j < size; j++ {
			// Get the current bit to be set
			bit := (value >> (size - 1 - j)) & 1

			// Get the byte position and the bit position within that byte
			bytePos := offset / 8
			bitPos := 7 - (offset % 8)

			// Clear the bit at the current position in the byte (using AND with a mask)
			data[bytePos] &^= (1 << bitPos)

			// Set the bit at the current position (using OR)
			data[bytePos] |= (byte(bit) << bitPos)

			// Move the offset to the next bit
			offset++
		}

		result = append(result, "OK")

	case "INCRBY":
		// Increment the value at the specified offset
		if len(list) < 2 {
			return nil, fmt.Errorf("-ERR invalid number of arguments for BITFIELD INCRBY operation")
		}
		incrValue := int64(list[1])

		// Resize the bitfield if necessary
		if offset+size > len(data)*8 {
			newSize := (offset + size + 7) / 8
			newData := make([]byte, newSize)
			copy(newData, data)
			data = newData
			r.Store[key] = data
		}

		// Increment the value at the offset
		value := int64(0)
		for j := 0; j < size; j++ {
			bit := (data[offset/8] >> (7 - offset%8)) & 1
			value |= int64(bit) << (size - 1 - j)
			offset++
		}

		value += incrValue

		// Store the incremented value back into the bitfield
		offset -= size
		for j := 0; j < size; j++ {
			bit := (value >> (size - 1 - j)) & 1
			bytePos := offset / 8
			bitPos := 7 - (offset % 8)

			// Clear the bit at the current position in the byte (using AND with a mask)
			data[bytePos] &^= (1 << bitPos)

			// Set the bit at the current position (using OR)
			data[bytePos] |= (byte(bit) << bitPos)

			offset++
		}

		result = append(result, value)

	default:
		return nil, fmt.Errorf("-ERR unknown BITFIELD operation %s", op)
	}

	return result, nil
}
