package storage

import (
	"errors"
	"fmt"
	"math"
)

// SetBitfield sets values in the bitfield at a given offset.
func (r *Tealis) SetBitfield(key string, bitType string, offset int, value int) error {
	r.Mu.Lock()
	defer r.Mu.Unlock()

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
func (r *Tealis) GetBitfield(key string, bitType string, offset int) (int, error) {
	r.Mu.RLock()
	defer r.Mu.RUnlock()

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
func (r *Tealis) IncrByBitfield(key string, bitType string, offset int, increment int) (int, error) {
	r.Mu.Lock()
	defer r.Mu.Unlock()

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

	// Check for overflow based on bit type
	maxValue := getMaxValueForBitType(bitType)
	minValue := getMinValueForBitType(bitType)

	if newValue > maxValue || newValue < minValue {
		return 0, fmt.Errorf("overflow: value %d exceeds the range for %s bitfield", newValue, bitType)
	}

	// Update the bitfield
	if err := r.setBitfieldValue(bitfield, offset, newValue, getBitfieldSize(bitType), key); err != nil {
		return 0, err
	}

	return newValue, nil
}

// getMaxValueForBitType returns the maximum value for a given bitfield type.
func getMaxValueForBitType(bitType string) int {
	switch bitType {
	case "i8":
		return 127 // Maximum value for signed 8-bit (two's complement)
	case "u8":
		return 255 // Maximum value for unsigned 8-bit
	case "i16":
		return 32767 // Maximum value for signed 16-bit
	case "u16":
		return 65535 // Maximum value for unsigned 16-bit
	// Add more types as needed
	default:
		return 0 // Default case for unsupported bit types
	}
}

// getMinValueForBitType returns the minimum value for a given bitfield type.
func getMinValueForBitType(bitType string) int {
	switch bitType {
	case "i8":
		return -128 // Minimum value for signed 8-bit (two's complement)
	case "u8":
		return 0 // Minimum value for unsigned 8-bit
	case "i16":
		return -32768 // Minimum value for signed 16-bit
	case "u16":
		return 0 // Minimum value for unsigned 16-bit
	// Add more types as needed
	default:
		return 0 // Default case for unsupported bit types
	}
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
func (r *Tealis) setBitfieldValue(bitfield []byte, offset int, value int, size int, key string) error {
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

	// Handle signed bitfield values (e.g., i8)
	if size == 8 && value&(1<<(size-1)) != 0 {
		// Two's complement sign extension for signed 8-bit integers
		value -= 1 << size
	}

	// Handle unsigned 16-bit values (e.g., u16)
	if size == 16 && value < 0 {
		// For unsigned 16-bit, we don't expect negative numbers
		// (i.e., the bitfield should always represent a positive number)
		return value & 0xFFFF, nil // Mask out the negative sign bits
	}

	// Return the value as is if it's unsigned
	return value, nil
}
