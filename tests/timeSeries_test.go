package storage

import (
	"tealis/internal/storage"
	"testing"
	"time"
)

func TestTimeSeries(t *testing.T) {
	// Initialize a new Redis clone instance
	r := storage.NewRedisClone()

	// Test TS.CREATE
	err := r.TSCreate("temperature", "avg")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Test that attempting to create an existing time series fails
	err = r.TSCreate("temperature", "avg")
	if err == nil {
		t.Fatalf("Expected error when creating existing time series, got nil")
	}

	// Test TS.ADD
	err = r.TSAdd("temperature", time.Now(), 22.5)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Add more data points
	err = r.TSAdd("temperature", time.Now().Add(time.Minute), 23.0)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	err = r.TSAdd("temperature", time.Now().Add(2*time.Minute), 21.5)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Test TS.GET to get the latest data point
	latest, err := r.TSGet("temperature")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if latest.Value != 21.5 {
		t.Fatalf("Expected latest value 21.5, got %v", latest.Value)
	}

	// Test TS.RANGE to get data points in a time range
	start := time.Now().Add(-5 * time.Minute)
	end := time.Now().Add(5 * time.Minute)
	data, err := r.TSRange("temperature", start, end)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if len(data) != 3 {
		t.Fatalf("Expected 3 data points, got %d", len(data))
	}

	// Test downsampling (avg)
	downsampledData, err := r.DownSample("temperature", start, end, time.Minute, "avg")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if len(downsampledData) != 3 {
		t.Fatalf("Expected 3 downsampled data points, got %d", len(downsampledData))
	}

	// Test aggregation values
	if downsampledData[0].Value != 22.5 {
		t.Fatalf("Expected aggregated value 22.5, got %v", downsampledData[0].Value)
	}
	if downsampledData[1].Value != 23.0 {
		t.Fatalf("Expected aggregated value 23.0, got %v", downsampledData[1].Value)
	}
	if downsampledData[2].Value != 21.5 {
		t.Fatalf("Expected aggregated value 21.5, got %v", downsampledData[2].Value)
	}
}

func TestTimeSeriesDownsamplingMinMax(t *testing.T) {
	r := storage.NewRedisClone()

	// Create time series with different aggregation methods
	err := r.TSCreate("temperature_avg", "avg")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	err = r.TSCreate("temperature_min", "min")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	err = r.TSCreate("temperature_max", "max")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Add data points
	r.TSAdd("temperature_avg", time.Now(), 22.5)
	r.TSAdd("temperature_avg", time.Now().Add(time.Minute), 23.0)
	r.TSAdd("temperature_avg", time.Now().Add(2*time.Minute), 21.5)

	r.TSAdd("temperature_min", time.Now(), 22.5)
	r.TSAdd("temperature_min", time.Now().Add(time.Minute), 23.0)
	r.TSAdd("temperature_min", time.Now().Add(2*time.Minute), 21.5)

	r.TSAdd("temperature_max", time.Now(), 22.5)
	r.TSAdd("temperature_max", time.Now().Add(time.Minute), 23.0)
	r.TSAdd("temperature_max", time.Now().Add(2*time.Minute), 21.5)

	// Test downsampling avg aggregation
	start := time.Now().Add(-5 * time.Minute)
	end := time.Now().Add(5 * time.Minute)
	downsampledAvg, err := r.DownSample("temperature_avg", start, end, time.Minute, "avg")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if len(downsampledAvg) != 3 {
		t.Fatalf("Expected 3 downsampled data points for avg, got %d", len(downsampledAvg))
	}

	// Test downsampling min aggregation
	downsampledMin, err := r.DownSample("temperature_min", start, end, time.Minute, "min")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if len(downsampledMin) != 3 {
		t.Fatalf("Expected 3 downsampled data points for min, got %d", len(downsampledMin))
	}

	// Test downsampling max aggregation
	downsampledMax, err := r.DownSample("temperature_max", start, end, time.Minute, "max")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if len(downsampledMax) != 3 {
		t.Fatalf("Expected 3 downsampled data points for max, got %d", len(downsampledMax))
	}

	// Check aggregation results for min and max
	if downsampledMin[0].Value != 22.5 {
		t.Fatalf("Expected min value 22.5, got %v", downsampledMin[0].Value)
	}
	if downsampledMax[1].Value != 23.0 {
		t.Fatalf("Expected max value 23.0, got %v", downsampledMax[1].Value)
	}
}
