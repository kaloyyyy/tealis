package storage

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
)

type RedisClone struct {
	Mu                sync.RWMutex
	Store             map[string]interface{} // Store can hold any data type (string, list, etc.)
	Expiries          map[string]time.Time
	transactions      map[string][]string               // Store queued commands for each transaction
	pubsubSubscribers map[string]map[string]chan string // channel -> clientID -> message channel
	ClientConnections map[string]interface{}            // clientID -> connection (can be net.Conn or *websocket.Conn)
	mockClients       map[string]*MockClientConnection

	// Persistence options
	AofFile       *os.File // Append-Only File
	aofFilePath   string   // Path to the AOF file
	enableAOF     bool     // Flag to enable/disable AOF
	snapshotPath  string   // Path to the snapshot file
	snapshotMutex sync.Mutex
}

func NewRedisClone(aofFilePath, snapshotPath string, enableAOF bool) *RedisClone {
	r := &RedisClone{
		Store:             make(map[string]interface{}),
		Expiries:          make(map[string]time.Time),
		transactions:      make(map[string][]string),
		pubsubSubscribers: make(map[string]map[string]chan string),
		ClientConnections: make(map[string]interface{}),
		mockClients:       make(map[string]*MockClientConnection),
		aofFilePath:       aofFilePath,
		enableAOF:         enableAOF,
		snapshotPath:      snapshotPath,
	}

	// Open AOF file if enabled
	if enableAOF {
		// Ensure the directory for the snapshot exists
		dir := r.snapshotPath
		if err := os.MkdirAll(dir, 0755); err != nil {
			_ = fmt.Errorf("failed to create directory for snapshot: %w, path: %s", err, dir)
		}
		aofFile, err := os.OpenFile(snapshotPath+"/"+aofFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Fatalf("Failed to open AOF file: %v", err)
		}
		r.AofFile = aofFile
		r.loadAOF()
	}
	// Start cleanup after loading AOF
	go r.StartCleanup(context.Background())
	return r
}

type MockClientConnection struct {
	Outbox chan string // Simulates the client's ability to receive messages
}

// MULTI starts a transaction.
func (r *RedisClone) MULTI(clientID string) {
	r.Mu.Lock()
	defer r.Mu.Unlock()
	r.transactions[clientID] = []string{} // Start a new transaction for the client
}

// EXEC executes all the queued commands in a transaction.
// EXEC executes all the queued commands in a transaction and returns their results.
func (r *RedisClone) EXEC(clientID string) string {
	r.Mu.Lock()
	defer r.Mu.Unlock()

	// Check if there are queued commands for this client
	if commands, ok := r.transactions[clientID]; ok {
		var response []string // This will hold the responses for each command

		// Iterate through each queued command and process it
		for _, cmd := range commands {
			// Split the command into parts (assuming it's space-separated, e.g., SET key value)
			parts := strings.Fields(cmd)

			// Process the command and capture the response
			responseStr := ProcessCommand(parts, r, clientID)
			response = append(response, responseStr)

			// Log the command execution
			log.Printf("Executing command in transaction: %s, Response: %s", cmd, responseStr)
		}

		// After executing, clear the transaction for the client
		delete(r.transactions, clientID)

		// Return all responses in the transaction, joined by a newline
		return strings.Join(response, "\r\n") + "\r\n"
	}

	// If no transaction was started, return an error
	return "-ERR No transaction started\r\n"
}

// DISCARD discards all the queued commands in the transaction.
func (r *RedisClone) DISCARD(clientID string) {
	r.Mu.Lock()
	defer r.Mu.Unlock()
	delete(r.transactions, clientID)
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
				r.Mu.Lock()
				for key, expiry := range r.Expiries {
					if now.After(expiry) {
						delete(r.Store, key)
						delete(r.Expiries, key)
					}
				}
				r.Mu.Unlock()
			}
		}
	}()
}

// EX sets the expiry time for a given key.
// The duration argument is the time duration after which the key will expire.
func (r *RedisClone) EX(key string, duration time.Duration) {
	r.Mu.Lock()
	defer r.Mu.Unlock()

	expiryTime := time.Now().Add(duration)
	r.Expiries[key] = expiryTime
}

// AppendToAOF writes a command to the AOF log.
func (r *RedisClone) AppendToAOF(command string) {
	if !r.enableAOF || r.AofFile == nil {
		return
	}
	r.Mu.Lock()
	defer r.Mu.Unlock()
	_, err := r.AofFile.WriteString(command + "\n")
	if err != nil {
		log.Printf("Error writing to AOF: %v", err)
	}
}

// SaveSnapshot creates a snapshot of the current state of the database.
func (r *RedisClone) SaveSnapshot() error {
	r.snapshotMutex.Lock()
	defer r.snapshotMutex.Unlock()

	r.Mu.RLock()
	defer r.Mu.RUnlock()

	// Validate snapshot path
	if r.snapshotPath == "" {
		return fmt.Errorf("snapshot path is not set")
	}

	// Ensure the directory for the snapshot exists
	dir := r.snapshotPath
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory for snapshot: %w, path: %s", err, dir)
	}

	// Serialize the state to a JSON file
	file, err := os.Create(dir + "/" + r.aofFilePath)
	if err != nil {
		return fmt.Errorf("failed to create snapshot file: %w, file loc: %s", err, r.snapshotPath)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	state := map[string]interface{}{
		"store":    r.Store,
		"expiries": r.Expiries,
	}
	if err := encoder.Encode(state); err != nil {
		return fmt.Errorf("failed to encode snapshot: %w", err)
	}

	return nil
}

// LoadSnapshot loads the state from a snapshot file.
func (r *RedisClone) LoadSnapshot() error {
	r.snapshotMutex.Lock()
	defer r.snapshotMutex.Unlock()

	file, err := os.Open(r.snapshotPath + "/" + r.aofFilePath)
	if err != nil {
		return fmt.Errorf("failed to open snapshot file: %w", err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	state := map[string]interface{}{}
	if err := decoder.Decode(&state); err != nil {
		return fmt.Errorf("failed to decode snapshot: %w", err)
	}

	r.Mu.Lock()
	defer r.Mu.Unlock()

	// Restore state
	if store, ok := state["store"].(map[string]interface{}); ok {
		r.Store = store
	}
	if expiries, ok := state["expiries"].(map[string]interface{}); ok {
		r.Expiries = make(map[string]time.Time)
		for k, v := range expiries {
			if t, err := time.Parse(time.RFC3339, v.(string)); err == nil {
				r.Expiries[k] = t
			}
		}
	}

	return nil
}

// LoadAOF replays commands from the AOF file to restore the database state.
func (r *RedisClone) loadAOF() {
	r.Mu.Lock()
	defer r.Mu.Unlock()

	file, err := os.Open(r.snapshotPath + "/" + r.aofFilePath)
	if err != nil {
		log.Printf("Error loading AOF file: %v", err)
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		command := scanner.Text()
		parts := strings.Fields(command)
		if len(parts) > 0 {
			ProcessCommand(parts, r, "AOFLoader")
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("Error reading AOF file: %v", err)
	}
}

func (r *RedisClone) ZAdd(key string, score float64, member string) int {
	r.Mu.Lock()
	defer r.Mu.Unlock()

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
	r.Mu.RLock()
	defer r.Mu.RUnlock()

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
	r.Mu.RLock()
	defer r.Mu.RUnlock()

	if val, exists := r.Store[key]; exists {
		if ss, ok := val.(*SortedSet); ok {
			return ss.ZRank(member)
		}
		panic("WRONGTYPE Operation against a key holding the wrong kind of value")
	}
	return -1 // Key does not exist
}

func (r *RedisClone) ZRem(key string, member string) bool {
	r.Mu.Lock()
	defer r.Mu.Unlock()

	if val, exists := r.Store[key]; exists {
		if ss, ok := val.(*SortedSet); ok {
			return ss.ZRem(member)
		}
		panic("WRONGTYPE Operation against a key holding the wrong kind of value")
	}
	return false // Key does not exist
}

func (r *RedisClone) ZRangeByScore(key string, min, max float64) []string {
	r.Mu.RLock()
	defer r.Mu.RUnlock()

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

func (r *RedisClone) GetClientConnection(clientID string) (interface{}, error) {
	r.Mu.RLock()
	defer r.Mu.RUnlock()

	conn, exists := r.ClientConnections[clientID]
	if !exists {
		return nil, fmt.Errorf("client %s not found", clientID)
	}
	return conn, nil
}

func (r *RedisClone) StartSnapshotScheduler(ctx context.Context, interval time.Duration) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(interval):
				if err := r.SaveSnapshot(); err != nil {
					log.Printf("Error saving snapshot: %v", err)
				}
			}
		}
	}()
}
