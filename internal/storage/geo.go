package storage

import (
	"math"
	"sort"
)

func (r *Tealis) GEOAdd(key string, lat, lon float64, member string) {
	r.Mu.Lock()
	defer r.Mu.Unlock()

	// Check if the key exists and is a GeoSet
	if val, exists := r.Store[key]; exists {
		if geo, ok := val.(*GeoSet); ok {
			geo.Add(member, lat, lon)
			return
		}
		panic("WRONGTYPE Operation against a key holding the wrong kind of value")
	}

	// If key doesn't exist, create a new GeoSet
	geo := NewGeoSet()
	geo.Add(member, lat, lon)
	r.Store[key] = geo
}

func (r *Tealis) GEODist(key, member1, member2 string) float64 {
	r.Mu.RLock()
	defer r.Mu.RUnlock()

	if val, exists := r.Store[key]; exists {
		if geo, ok := val.(*GeoSet); ok {
			loc1, exists1 := geo.Locations[member1]
			loc2, exists2 := geo.Locations[member2]
			if exists1 && exists2 {
				return geo.Distance(loc1.Latitude, loc1.Longitude, loc2.Latitude, loc2.Longitude)
			}
			panic("One or both members do not exist")
		}
		panic("WRONGTYPE Operation against a key holding the wrong kind of value")
	}
	panic("Key does not exist")
}

func (r *Tealis) GEOSearch(key string, lat, lon, radius float64) []string {
	r.Mu.RLock()
	defer r.Mu.RUnlock()

	if val, exists := r.Store[key]; exists {
		if geo, ok := val.(*GeoSet); ok {
			return geo.SearchByRadius(lat, lon, radius)
		}
		panic("WRONGTYPE Operation against a key holding the wrong kind of value")
	}
	return nil // Key does not exist
}

// GeoLocation represents a geographic location.
type GeoLocation struct {
	Latitude  float64
	Longitude float64
	Name      string
}

// GeoSet represents a geospatial sorted set.
type GeoSet struct {
	Locations map[string]GeoLocation
	Sorted    []string // Maintain sorted keys for proximity searches.
}

// NewGeoSet initializes a new GeoSet.
func NewGeoSet() *GeoSet {
	return &GeoSet{
		Locations: make(map[string]GeoLocation),
		Sorted:    []string{},
	}
}

// Add adds a new location or updates an existing one.
func (g *GeoSet) Add(name string, lat, lon float64) {
	location := GeoLocation{
		Latitude:  lat,
		Longitude: lon,
		Name:      name,
	}
	g.Locations[name] = location
	g.Sorted = append(g.Sorted, name)
	sort.Strings(g.Sorted)
}

// Distance calculates the haversine distance between two coordinates in kilometers.
func (g *GeoSet) Distance(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371 // Earth radius in km
	dLat := (lat2 - lat1) * (math.Pi / 180)
	dLon := (lon2 - lon1) * (math.Pi / 180)
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*(math.Pi/180))*math.Cos(lat2*(math.Pi/180))*
			math.Sin(dLon/2)*math.Sin(dLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return R * c
}

// SearchByRadius finds locations within a radius from a point.
func (g *GeoSet) SearchByRadius(lat, lon, radius float64) []string {
	results := []string{}
	for _, loc := range g.Locations {
		dist := g.Distance(lat, lon, loc.Latitude, loc.Longitude)
		if dist <= radius {
			results = append(results, loc.Name)
		}
	}
	return results
}
