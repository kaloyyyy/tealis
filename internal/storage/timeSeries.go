package storage

import (
	"fmt"
	"math"
	"sort"
	"sync"
	"time"
)

// TimeSeries stores a list of time series data points.
type TimeSeries struct {
	mu          sync.RWMutex
	Points      []DataPoint // Sorted list of data points (timestamp, value)
	aggregation string      // Aggregation method for downsampling
}

// DataPoint represents a time series data point (timestamp, value).
type DataPoint struct {
	Timestamp time.Time
	Value     float64
}

// NewTimeSeries initializes a new time series.
func NewTimeSeries() *TimeSeries {
	return &TimeSeries{
		Points: []DataPoint{},
	}
}

// TSCreate TS.CREATE creates a new time series.
func (r *Tealis) TSCreate(key string, aggregation string) error {
	r.Mu.Lock()
	defer r.Mu.Unlock()

	// Check if the key already exists
	if _, exists := r.Store[key]; exists {
		return fmt.Errorf("time series %s already exists", key)
	}

	// Create a new time series with specified aggregation
	r.Store[key] = NewTimeSeries()
	ts := r.Store[key].(*TimeSeries)
	ts.aggregation = aggregation
	return nil
}

// TSAdd TS.ADD adds a new data point to the time series.
func (r *Tealis) TSAdd(key string, timestamp time.Time, value float64) error {
	r.Mu.Lock()
	defer r.Mu.Unlock()

	// Find the time series for the given key
	ts, exists := r.Store[key].(*TimeSeries)
	if !exists {
		return fmt.Errorf("time series %s not found", key)
	}

	// Add the new data point
	ts.mu.Lock()
	ts.Points = append(ts.Points, DataPoint{Timestamp: timestamp, Value: value})
	sort.Slice(ts.Points, func(i, j int) bool {
		return ts.Points[i].Timestamp.Before(ts.Points[j].Timestamp)
	})
	ts.mu.Unlock()

	return nil
}

// TSRange TS.RANGE returns the time series data points within the specified time range.
func (r *Tealis) TSRange(key string, start, end time.Time) ([]DataPoint, error) {
	r.Mu.RLock()
	defer r.Mu.RUnlock()

	// Find the time series for the given key
	ts, exists := r.Store[key].(*TimeSeries)
	if !exists {
		return nil, fmt.Errorf("time series %s not found", key)
	}

	// Filter data points within the specified range
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	var result []DataPoint
	for _, dp := range ts.Points {
		if dp.Timestamp.After(start) && dp.Timestamp.Before(end) {
			result = append(result, dp)
		}
	}
	return result, nil
}

// TSGet TS.GET retrieves the latest data point in the time series.
func (r *Tealis) TSGet(key string) (DataPoint, error) {
	r.Mu.RLock()
	defer r.Mu.RUnlock()

	// Find the time series for the given key
	ts, exists := r.Store[key].(*TimeSeries)
	if !exists {
		return DataPoint{}, fmt.Errorf("time series %s not found", key)
	}

	// Return the most recent data point
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	if len(ts.Points) == 0 {
		return DataPoint{}, fmt.Errorf("no data points in time series %s", key)
	}

	return ts.Points[len(ts.Points)-1], nil
}

// DownSample performs aggregation on the time series data.
func (r *Tealis) DownSample(key string, start, end time.Time, interval time.Duration, method string) ([]DataPoint, error) {
	r.Mu.RLock()
	defer r.Mu.RUnlock()

	// Find the time series for the given key
	ts, exists := r.Store[key].(*TimeSeries)
	if !exists {
		return nil, fmt.Errorf("time series %s not found", key)
	}

	// Filter data points within the specified range
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	var result []DataPoint
	var currentIntervalStart time.Time
	var currentValues []float64

	for _, dp := range ts.Points {
		if dp.Timestamp.After(start) && dp.Timestamp.Before(end) {
			intervalIndex := dp.Timestamp.Truncate(interval)
			if currentIntervalStart.IsZero() {
				currentIntervalStart = intervalIndex
			}

			if intervalIndex == currentIntervalStart {
				currentValues = append(currentValues, dp.Value)
			} else {
				aggregatedValue := aggregate(currentValues, method)
				result = append(result, DataPoint{Timestamp: currentIntervalStart, Value: aggregatedValue})
				currentIntervalStart = intervalIndex
				currentValues = []float64{dp.Value}
			}
		}
	}

	// Aggregate the last interval if it contains data
	if len(currentValues) > 0 {
		aggregatedValue := aggregate(currentValues, method)
		result = append(result, DataPoint{Timestamp: currentIntervalStart, Value: aggregatedValue})
	}

	return result, nil
}

// Aggregate applies the specified aggregation method (avg, tsMin, tsMax).
func aggregate(values []float64, method string) float64 {
	switch method {
	case "avg":
		var sum float64
		for _, v := range values {
			sum += v
		}
		return sum / float64(len(values))
	case "min":
		return tsMin(values)
	case "max":
		return tsMax(values)
	default:
		return 0
	}
}

func tsMin(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	minVal := math.Inf(1)
	for _, v := range values {
		if v < minVal {
			minVal = v
		}
	}
	return minVal
}

func tsMax(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	maxVal := math.Inf(-1)
	for _, v := range values {
		if v > maxVal {
			maxVal = v
		}
	}
	return maxVal
}
