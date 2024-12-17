package storage_test

import (
	"os"
	"tealis/internal/storage"
	"testing"
)

func TestGeoCommands(t *testing.T) {
	// Setup
	aofFilePath := "test.aof"
	defer os.Remove(aofFilePath) // Clean up the test AOF file

	// Initialize a RedisClone instance with AOF enabled
	store := storage.NewRedisClone(aofFilePath, "", true)

	// Test GEOAdd
	t.Run("GEOAdd", func(t *testing.T) {
		store.GEOAdd("geoKey", 13.361389, 38.115556, "Palermo")
		store.GEOAdd("geoKey", 15.087269, 37.502669, "Catania")

		geoSet := store.Store["geoKey"].(*storage.GeoSet)
		if len(geoSet.Locations) != 2 {
			t.Errorf("Expected 2 locations, got %d", len(geoSet.Locations))
		}
		if geoSet.Locations["Palermo"].Latitude != 13.361389 {
			t.Errorf("Palermo latitude mismatch: expected 13.361389, got %f", geoSet.Locations["Palermo"].Latitude)
		}
		if geoSet.Locations["Catania"].Longitude != 37.502669 {
			t.Errorf("Catania longitude mismatch: expected 37.502669, got %f", geoSet.Locations["Catania"].Longitude)
		}
	})

	// Test GEODist
	t.Run("GEODist", func(t *testing.T) {
		store.GEOAdd("geoKey", 13.361389, 38.115556, "Palermo")
		store.GEOAdd("geoKey", 15.087269, 37.502669, "Catania")

		dist := store.GEODist("geoKey", "Palermo", "Catania")
		expectedDist := 202.9598 // Example distance in km
		if !closeEnough(dist, expectedDist, 0.0001) {
			t.Errorf("Expected distance %.4f, got %.4f", expectedDist, dist)
		}
	})

	// Test GEOSearch
	t.Run("GEOSearch", func(t *testing.T) {
		store.GEOAdd("geoKey", 13.361389, 38.115556, "Palermo")
		store.GEOAdd("geoKey", 15.087269, 37.502669, "Catania")
		store.GEOAdd("geoKey", 40.0, 38.0, "AnotherCity")

		results := store.GEOSearch("geoKey", 13.361389, 38.115556, 300)
		expectedResults := []string{"Palermo", "Catania"}
		if !compareStringSlices(results, expectedResults) {
			t.Errorf("Expected results %v, got %v", expectedResults, results)
		}
	})
}

// Helper function to compare float values with a tolerance
func closeEnough(a, b, tolerance float64) bool {
	return (a-b) < tolerance && (b-a) < tolerance
}

// Helper function to compare string slices
func compareStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
