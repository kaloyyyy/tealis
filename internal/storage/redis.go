package storage

import (
	"context"
	"math"
	"sort"
	"sync"
	"time"
)

// RedisClone represents the basic structure for a Redis-like store.
type RedisClone struct {
	mu       sync.RWMutex
	Store    map[string]interface{} // Store can hold any data type (string, list, etc.)
	expiries map[string]time.Time
}

// NewRedisClone initializes a new RedisClone instance.
func NewRedisClone() *RedisClone {
	return &RedisClone{
		Store:    make(map[string]interface{}),
		expiries: make(map[string]time.Time),
	}
}

// StartCleanup periodically cleans expired keys.
func (r *RedisClone) StartCleanup(ctx context.Context) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Second):
				now := time.Now()
				r.mu.Lock()
				for key, expiry := range r.expiries {
					if now.After(expiry) {
						delete(r.Store, key)
						delete(r.expiries, key)
					}
				}
				r.mu.Unlock()
			}
		}
	}()
}

func (r *RedisClone) ZAdd(key string, score float64, member string) int {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if the key exists and is a sorted set
	if val, exists := r.Store[key]; exists {
		if ss, ok := val.(*SortedSet); ok {
			ss.ZAdd(member, score)
			return 1
		}
		panic("WRONGTYPE Operation against a key holding the wrong kind of value")
	}

	// If key doesn't exist, create a new SortedSet
	ss := NewSortedSet()
	ss.ZAdd(member, score)
	r.Store[key] = ss
	return 1
}

func (r *RedisClone) ZRange(key string, start, end int) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Check if the key exists and is a sorted set
	if val, exists := r.Store[key]; exists {
		if ss, ok := val.(*SortedSet); ok {
			return ss.ZRange(start, end)
		}
		panic("WRONGTYPE Operation against a key holding the wrong kind of value")
	}
	return nil // Key does not exist
}

func (r *RedisClone) ZRank(key string, member string) int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if val, exists := r.Store[key]; exists {
		if ss, ok := val.(*SortedSet); ok {
			return ss.ZRank(member)
		}
		panic("WRONGTYPE Operation against a key holding the wrong kind of value")
	}
	return -1 // Key does not exist
}

func (r *RedisClone) ZRem(key string, member string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	if val, exists := r.Store[key]; exists {
		if ss, ok := val.(*SortedSet); ok {
			return ss.ZRem(member)
		}
		panic("WRONGTYPE Operation against a key holding the wrong kind of value")
	}
	return false // Key does not exist
}

func (r *RedisClone) ZRangeByScore(key string, min, max float64) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if val, exists := r.Store[key]; exists {
		if ss, ok := val.(*SortedSet); ok {
			return ss.ZRangeByScore(min, max)
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
