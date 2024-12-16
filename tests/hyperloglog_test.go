package storage

import (
	"tealis/internal/storage"
	"testing"
)

func TestHyperLogLog_AddAndCount(t *testing.T) {
	// Initialize a new HyperLogLog with precision 14
	hll := storage.NewHyperLogLog(14)

	// Add distinct elements
	hll.Add("apple")
	hll.Add("banana")
	hll.Add("cherry")

	// Estimate the cardinality
	count := hll.Count()

	// The estimated count should be approximately 3
	if count < 2 || count > 4 {
		t.Errorf("Expected cardinality around 3, got %d", count)
	}

	// Add duplicate elements
	hll.Add("apple")
	hll.Add("banana")

	// Re-estimate the cardinality
	count = hll.Count()

	// The count should remain approximately 3
	if count < 2 || count > 4 {
		t.Errorf("Expected cardinality around 3 after adding duplicates, got %d", count)
	}
}

func TestHyperLogLog_Empty(t *testing.T) {
	// Initialize a new HyperLogLog with precision 14
	hll := storage.NewHyperLogLog(14)

	// Count without adding elements
	count := hll.Count()

	// The estimated count should be 0
	if count != 0 {
		t.Errorf("Expected cardinality 0 for an empty HyperLogLog, got %d", count)
	}
}

func TestHyperLogLog_Merge(t *testing.T) {
	// Initialize two HyperLogLogs
	hll1 := storage.NewHyperLogLog(14)
	hll2 := storage.NewHyperLogLog(14)

	// Add elements to the first HyperLogLog
	hll1.Add("apple")
	hll1.Add("banana")

	// Add elements to the second HyperLogLog
	hll2.Add("cherry")
	hll2.Add("date")

	// Merge the two HyperLogLogs
	hll1.Merge(hll2)

	// Estimate the cardinality of the merged HyperLogLog
	count := hll1.Count()

	// The estimated count should be approximately 4
	if count < 3 || count > 5 {
		t.Errorf("Expected cardinality around 4 after merge, got %d", count)
	}
}

func TestHyperLogLog_MergeDifferentRegisters(t *testing.T) {
	// Initialize two HyperLogLogs with different precision
	hll1 := storage.NewHyperLogLog(14)
	hll2 := storage.NewHyperLogLog(15) // Different precision

	// Attempting to merge should panic
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected panic when merging HyperLogLogs with different precision")
		}
	}()
	hll1.Merge(hll2)
}
