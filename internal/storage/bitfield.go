package storage

import (
	"errors"
	"math"
)

// SetBitfield sets values in the bitfield at a given offset.
func (r *RedisClone) SetBitfield(key string, bitType string, offset int, value int) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Initialize the bitfield if it doesn't exist.
	if _, exists := r.Store[key]; !exists {
		r.Store[key] = make([]byte, 0)
	}

	bitfield := r.Store[key].([]byte)

	switch bitType {
	case "i8":
		// Initialize the bitfield if it doesn't exist.
		if _, exists := r.Store[key]; !exists {
			// Initialize with a signed 8-bit integer (e.g., -128)
			// Convert signed int8 to byte by directly casting
			val := int8(-128)
			r.Store[key] = []byte{byte(val)} // Store the signed int8 as byte (two's complement)
		}
		if value < math.MinInt8 || value > math.MaxInt8 {
			return errors.New("value out of range for i8")
		}
		// Set the value for an 8-bit signed integer.
		return r.setBitfieldValue(bitfield, offset, value, 8, key)
	case "u16":
		// Initialize the bitfield if it doesn't exist.
		if _, exists := r.Store[key]; !exists {
			// Initialize with an unsigned 16-bit integer (e.g., 65535)
			// Store the uint16 value as two bytes (little-endian or big-endian depending on your need)
			val := uint16(65535)
			r.Store[key] = []byte{byte(val & 0xFF), byte(val >> 8)} // Little-endian format
		}

		if value < 0 || value > math.MaxUint16 {
			return errors.New("value out of range for u16")
		}
		// Set the value for a 16-bit unsigned integer.
		return r.setBitfieldValue(bitfield, offset, value, 16, key)
	default:
		return errors.New("unsupported bitfield type")
	}
}

// GetBitfield gets the value stored at the given offset for a specified type.
func (r *RedisClone) GetBitfield(key string, bitType string, offset int) (int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if _, exists := r.Store[key]; !exists {
		return 0, errors.New("key not found")
	}

	bitfield := r.Store[key].([]byte)

	switch bitType {
	case "i8":
		return getBitfieldValue(bitfield, offset, 8)
	case "u16":
		return getBitfieldValue(bitfield, offset, 16)
	default:
		return 0, errors.New("unsupported bitfield type")
	}
}

// IncrByBitfield increments the value of the bitfield at a given offset by a specified increment.
func (r *RedisClone) IncrByBitfield(key string, bitType string, offset int, increment int) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.Store[key]; !exists {
		return 0, errors.New("key not found")
	}

	bitfield := r.Store[key].([]byte)

	// Get the current value
	currentValue, err := getBitfieldValue(bitfield, offset, getBitfieldSize(bitType))
	if err != nil {
		return 0, err
	}

	// Increment the value
	newValue := currentValue + increment

	// Update the bitfield
	if err := r.setBitfieldValue(bitfield, offset, newValue, getBitfieldSize(bitType), key); err != nil {
		return 0, err
	}

	return newValue, nil
}

// Helper function to determine the bitfield size based on the type
func getBitfieldSize(bitType string) int {
	switch bitType {
	case "i8":
		return 8
	case "u16":
		return 16
	default:
		return 0
	}
}

// Helper function to set a bitfield value at a specified offset
func (r *RedisClone) setBitfieldValue(bitfield []byte, offset int, value int, size int, key string) error {
	// Ensure the bitfield has enough space
	requiredBytes := (offset + size + 7) / 8 // Round up to ensure enough space
	if len(bitfield) < requiredBytes {
		bitfield = append(bitfield, make([]byte, requiredBytes-len(bitfield))...)
	}

	for i := 0; i < size; i++ {
		// Set each bit in the bitfield based on the value
		bit := (value >> (size - 1 - i)) & 1
		byteIndex := (offset + i) / 8
		bitIndex := (offset + i) % 8
		if bit == 1 {
			bitfield[byteIndex] |= (1 << (7 - bitIndex)) // Set bit to 1
		} else {
			bitfield[byteIndex] &= ^(1 << (7 - bitIndex)) // Set bit to 0
		}
	}
	r.Store[key] = bitfield
	print(bitfield)
	return nil
}

// Helper function to get the value from a bitfield at a specified offset
func getBitfieldValue(bitfield []byte, offset int, size int) (int, error) {
	// Ensure that size is valid
	if size <= 0 || size > 64 {
		return 0, errors.New("invalid bitfield size")
	}

	// Initialize the value to 0
	var value int

	// Loop through each bit in the specified size
	for i := 0; i < size; i++ {
		// Determine the byte index and bit index
		byteIndex := (offset + i) / 8
		bitIndex := (offset + i) % 8

		// Check if byteIndex is out of bounds
		if byteIndex >= len(bitfield) {
			return 0, errors.New("offset out of range")
		}

		// Extract the bit and shift it into the correct position in the value
		bit := (bitfield[byteIndex] >> (7 - bitIndex)) & 1

		// Accumulate the bit into the result value by shifting it to the correct position
		value |= int(bit) << (size - 1 - i)
	}

	// If the value is negative (signed bitfield), perform sign extension
	if size > 0 && value&(1<<(size-1)) != 0 { // If the highest bit is set (negative value in signed bitfields)
		// Perform two's complement sign extension
		value -= 1 << size
	}

	return value, nil
}
