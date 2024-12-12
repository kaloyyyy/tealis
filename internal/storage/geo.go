package storage

func (r *RedisClone) GEOAdd(key string, lat, lon float64, member string) {
	r.mu.Lock()
	defer r.mu.Unlock()

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

func (r *RedisClone) GEODist(key, member1, member2 string) float64 {
	r.mu.RLock()
	defer r.mu.RUnlock()

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

func (r *RedisClone) GEOSearch(key string, lat, lon, radius float64) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if val, exists := r.Store[key]; exists {
		if geo, ok := val.(*GeoSet); ok {
			return geo.SearchByRadius(lat, lon, radius)
		}
		panic("WRONGTYPE Operation against a key holding the wrong kind of value")
	}
	return nil // Key does not exist
}
