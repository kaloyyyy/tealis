package storage

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
)

func (r *Tealis) VectorSet(key string, vector []float64) string {
	r.Mu.Lock()
	defer r.Mu.Unlock()
	r.Store[key] = vector
	if r.enableAOF {
		r.AppendToAOF(fmt.Sprintf("VECTOR.SET %s %v", key, vector))
	}
	return "+OK\r\n"
}

func (r *Tealis) VectorGet(key string) string {
	r.Mu.RLock()
	defer r.Mu.RUnlock()
	if value, exists := r.Store[key]; exists {
		if vector, ok := value.([]float64); ok {
			response, _ := json.Marshal(vector)
			return fmt.Sprintf("$%d\r\n%s\r\n", len(response), response)
		}
		return "-ERR Key is not a vector\r\n"
	}
	return "-ERR Key not found\r\n"
}

func (r *Tealis) VectorSearch(query []float64, k int) string {
	r.Mu.RLock()
	defer r.Mu.RUnlock()

	type result struct {
		key      string
		distance float64
	}

	results := []result{}

	for key, value := range r.Store {
		if vector, ok := value.([]float64); ok {
			dist := CosineSimilarity(query, vector)
			results = append(results, result{key: key, distance: dist})
		}
	}

	// Sort by distance (smallest first)
	sort.Slice(results, func(i, j int) bool {
		return results[i].distance < results[j].distance
	})

	// Take the top-k results
	var response []string
	for i := 0; i < k && i < len(results); i++ {
		response = append(response, fmt.Sprintf("%s: %f", results[i].key, results[i].distance))
	}

	return strings.Join(response, "\r\n") + "\r\n"
}

func CosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) {
		return 1.0 // High distance for incompatible dimensions
	}

	var dot, normA, normB float64
	for i := range a {
		dot += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 1.0 // High distance for zero vectors
	}

	return 1 - dot/(math.Sqrt(normA)*math.Sqrt(normB))
}
