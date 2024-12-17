package storage

import (
	"errors"
	"hash/fnv"
	"math"
	"math/bits"
)

// HyperLogLog represents a HyperLogLog data structure.
type HyperLogLog struct {
	registers []uint8 // Registers to store the maximum rank
	m         uint32  // Number of registers
	alphaMM   float64 // Precomputed alpha * m^2
}

// NewHyperLogLog initializes a new HyperLogLog instance with the given precision.
// Precision determines the number of registers as 2^precision.
func NewHyperLogLog(precision uint8) *HyperLogLog {
	if precision < 4 || precision > 18 {
		panic("Precision must be between 4 and 18")
	}
	m := uint32(1 << precision) // Number of registers
	alphaMM := 0.7213 / (1 + 1.079/float64(m)) * float64(m*m)

	return &HyperLogLog{
		registers: make([]uint8, m),
		m:         m,
		alphaMM:   alphaMM,
	}
}

// Add inserts a value into the HyperLogLog structure.
func (hll *HyperLogLog) Add(value string) {
	// Hash the input value
	hash := fnv.New64a()
	hash.Write([]byte(value))
	x := hash.Sum64()

	// Extract the register index (log2(m) bits)
	registerIndex := uint32(x) & (hll.m - 1) // Mask to the correct number of bits

	// Count leading zeros in the remaining hash bits + 1
	w := x >> (64 - bits.Len32(hll.m))
	leadingZeros := bits.LeadingZeros64(w) + 1

	// Update the register with the maximum observed rank
	if hll.registers[registerIndex] < uint8(leadingZeros) {
		hll.registers[registerIndex] = uint8(leadingZeros)
	}
}

// Count estimates the cardinality of the dataset.
func (hll *HyperLogLog) Count() int64 {
	// Harmonic mean of the register values
	sum := 0.0
	for _, register := range hll.registers {
		sum += 1.0 / math.Pow(2.0, float64(register))
	}

	// Raw HyperLogLog estimate
	estimate := hll.alphaMM / sum

	// Small range correction (empty registers heuristic)
	emptyRegisters := 0
	for _, register := range hll.registers {
		if register == 0 {
			emptyRegisters++
		}
	}
	if emptyRegisters > 0 {
		estimate = float64(hll.m) * math.Log(float64(hll.m)/float64(emptyRegisters))
	}

	// Large range correction
	if estimate > float64(1<<32)/30.0 {
		estimate = -(float64(1<<32) * math.Log(1-estimate/float64(1<<32)))
	}

	return int64(estimate)
}

// Merge combines another HyperLogLog into this one.
func (hll *HyperLogLog) Merge(other *HyperLogLog) {
	if hll.m != other.m {
		panic("Cannot merge HyperLogLogs with different register sizes")
	}

	for i := range hll.registers {
		if other.registers[i] > hll.registers[i] {
			hll.registers[i] = other.registers[i]
		}
	}
}

func (r *RedisClone) PFAdd(key string, value string) error {
	r.Mu.Lock()
	defer r.Mu.Unlock()

	if val, exists := r.Store[key]; exists {
		if hll, ok := val.(*HyperLogLog); ok {
			hll.Add(value)
			return nil
		}
		return errors.New("WRONGTYPE Operation against a key holding the wrong kind of value")
	}

	// Create a new HyperLogLog if the key does not exist
	hll := NewHyperLogLog(14) // Default precision of 14
	hll.Add(value)
	r.Store[key] = hll
	return nil
}

func (r *RedisClone) PFCount(key string) (int64, error) {
	r.Mu.RLock()
	defer r.Mu.RUnlock()

	if val, exists := r.Store[key]; exists {
		if hll, ok := val.(*HyperLogLog); ok {
			return hll.Count(), nil
		}
		return 0, errors.New("WRONGTYPE Operation against a key holding the wrong kind of value")
	}
	return 0, errors.New("key does not exist") // Key does not exist
}

func (r *RedisClone) PFMerge(dest string, sources ...string) error {
	r.Mu.Lock()
	defer r.Mu.Unlock()

	var merged *HyperLogLog
	for _, source := range sources {
		if val, exists := r.Store[source]; exists {
			if hll, ok := val.(*HyperLogLog); ok {
				if merged == nil {
					merged = NewHyperLogLog(14) // Use the same precision
				}
				merged.Merge(hll)
			} else {
				return errors.New("WRONGTYPE Operation against a key holding the wrong kind of value")
			}
		} else {
			return errors.New("source key does not exist")
		}
	}

	r.Store[dest] = merged
	return nil
}
