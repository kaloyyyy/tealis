package storage

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
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
		dir := r.aofFilePath
		print(dir)
		if err := os.MkdirAll(dir, 0755); err != nil {
			_ = fmt.Errorf("failed to create directory for snapshot: %w, path: %s", err, dir)
		}
		aofFile, err := os.OpenFile(aofFilePath+"/aof.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Fatalf("Enable AOF Failed to open AOF file: %v", err)
		}
		r.AofFile = aofFile
		r.loadAOF()
	}
	// Start cleanup after loading AOF
	go r.StartCleanup(context.Background())
	return r
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
	// Check if AOF is enabled and AofFile is nil
	if !r.enableAOF || r.AofFile == nil {
		log.Printf("AOF is disabled or AofFile is nil, not appending command.")
		return
	}

	// Reopen the file if it's closed
	if err := r.ensureAOFFileOpen(); err != nil {
		log.Printf("Error ensuring AOF file is open: %v", err)
		return
	}

	// Write the command to the AOF file
	_, err := r.AofFile.WriteString(command + "\n")
	if err != nil {
		log.Printf("Error writing to AOF: %v", err)
		return
	}

	// Log successful write
	log.Printf("Command appended to AOF: %s", command)
}

// ensureAOFFileOpen ensures the AOF file is open.
func (r *RedisClone) ensureAOFFileOpen() error {
	// If AofFile is nil or closed, reopen the file
	if r.AofFile == nil || r.AofFileClosed() {
		aofFile, err := os.OpenFile(r.aofFilePath+"/aof.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return fmt.Errorf("failed to open AOF file: %v", err)
		}
		r.AofFile = aofFile
	}
	return nil
}

// AofFileClosed checks if the AOF file is closed.
func (r *RedisClone) AofFileClosed() bool {
	// Try writing to the file to check if it's closed
	_, err := r.AofFile.WriteString("")
	return err != nil
}

func (r *RedisClone) loadAOF() {
	r.Mu.RLock()
	file, err := os.Open(r.aofFilePath + "/aof.txt")
	if err != nil {
		log.Printf("Error loading AOF file: %v", err)
		r.Mu.RUnlock()
		return
	}
	r.Mu.RUnlock() // Release lock after opening file
	defer file.Close()

	var commands [][]string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		command := scanner.Text()
		parts := strings.Fields(command)
		if len(parts) > 0 {
			commands = append(commands, parts)
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("Error reading AOF file: %v", err)
		return
	}

	// Process commands without holding the lock
	for _, parts := range commands {
		ProcessCommand(parts, r, "AOFLoader")
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
	print(dir)
	if err := os.MkdirAll(dir, 0755); err != nil {
		_ = fmt.Errorf("failed to create directory for snapshot: %w, path: %s", err, dir)
	}
	// Create or open the snapshot file
	file, err := os.Create(r.snapshotPath + "/text.json")
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

	file, err := os.Open(r.snapshotPath + "/text.json")
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
