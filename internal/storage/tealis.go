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

type Tealis struct {
	Mu                sync.RWMutex
	Store             map[string]interface{} // Store can hold any data type (string, list, etc.)
	Expiries          map[string]time.Time
	Transactions      map[string][]string               // Store queued commands for each transaction
	pubsubSubscribers map[string]map[string]chan string // channel -> clientID -> message channel
	ClientConnections map[string]interface{}            // clientID -> connection (can be net.Conn or *websocket.Conn)
	mockClients       map[string]*MockClientConnection
	multi             bool
	// Persistence options
	AofFile       *os.File // Append-Only File
	aofFilePath   string   // Path to the AOF file
	enableAOF     bool     // Flag to enable/disable AOF
	snapshotPath  string   // Path to the snapshot file
	snapshotMutex sync.Mutex
}

func NewTealis(aofFilePath, snapshotPath string, enableAOF bool) *Tealis {
	r := &Tealis{
		Store:             make(map[string]interface{}),
		Expiries:          make(map[string]time.Time),
		Transactions:      make(map[string][]string),
		pubsubSubscribers: make(map[string]map[string]chan string),
		ClientConnections: make(map[string]interface{}),
		mockClients:       make(map[string]*MockClientConnection),
		multi:             false,
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
func (r *Tealis) MULTI(clientID string) {
	r.Mu.Lock()
	defer r.Mu.Unlock()
	r.Transactions[clientID] = []string{} // Start a new transaction for the client
	r.multi = true
}

func (r *Tealis) EXEC(clientID string) string {
	r.Mu.Lock()

	// Check if there are queued commands for this client
	commands, ok := r.Transactions[clientID]
	if !ok {
		r.Mu.Unlock()
		return "-ERR No transaction started\r\n"
	}

	// Copy commands to process outside the lock
	commandsToExecute := make([]string, len(commands))
	copy(commandsToExecute, commands)

	// Clear the transaction for the client
	delete(r.Transactions, clientID)

	r.Mu.Unlock() // Release the lock

	var response []string // This will hold the responses for each command
	// Return all responses in the transaction, joined by a newline
	r.multi = false
	// Process each command
	for _, cmd := range commandsToExecute {
		parts := strings.Fields(cmd) // Split the command into parts
		responseStr := ProcessCommand(parts, r, clientID)
		response = append(response, responseStr)

		// Log the command execution
		log.Printf("Executing command in transaction: %s, Response: %s", cmd, responseStr)
	}

	return strings.Join(response, "\r\n") + "\r\n"
}

// DISCARD discards all the queued commands in the transaction.
func (r *Tealis) DISCARD(clientID string) {
	r.Mu.Lock()
	defer r.Mu.Unlock()
	r.multi = false
	delete(r.Transactions, clientID)
}

// StartCleanup periodically cleans expired keys.
func (r *Tealis) StartCleanup(ctx context.Context) {
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
func (r *Tealis) EX(key string, duration time.Duration) {
	r.Mu.Lock()
	defer r.Mu.Unlock()

	expiryTime := time.Now().Add(duration)
	r.Expiries[key] = expiryTime
}

// AppendToAOF writes a command to the AOF log.
func (r *Tealis) AppendToAOF(command string) {
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
	log.Printf("fn: Command appended to AOF: %s", command)
}

// ensureAOFFileOpen ensures the AOF file is open.
func (r *Tealis) ensureAOFFileOpen() error {
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
func (r *Tealis) AofFileClosed() bool {
	// Try writing to the file to check if it's closed
	_, err := r.AofFile.WriteString("")
	return err != nil
}

func (r *Tealis) loadAOF() {
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

}

// RewriteAOF rewrites the AOF file to compact its contents and include only the current state.
func (r *Tealis) RewriteAOF() error {
	r.Mu.RLock()
	defer r.Mu.RUnlock()

	// Validate AOF file path
	if r.aofFilePath == "" {
		return fmt.Errorf("AOF file path is not set")
	}

	// Create a temporary file for the new AOF
	tempFilePath := r.aofFilePath + "/aof_rewrite.tmp"
	tempFile, err := os.Create(tempFilePath)
	if err != nil {
		return fmt.Errorf("failed to create temporary AOF file: %w", err)
	}
	defer tempFile.Close()

	// Write the current state to the temporary AOF file
	for key, value := range r.Store {
		// Serialize the value into a string command
		switch v := value.(type) {
		case string:
			_, err = tempFile.WriteString(fmt.Sprintf("SET %s %s\n", key, v))
		case []interface{}:
			_, err = tempFile.WriteString(fmt.Sprintf("RPUSH %s %s\n", key, formatListForAOF(v)))
		default:
			err = fmt.Errorf("unsupported type for key %s", key)
		}
		if err != nil {
			return fmt.Errorf("failed to write to temporary AOF file: %w", err)
		}

		// If the key has an expiry, add the expiry command
		if expiry, exists := r.Expiries[key]; exists {
			ttl := int64(time.Until(expiry).Seconds())
			if ttl > 0 {
				_, err = tempFile.WriteString(fmt.Sprintf("EX %s %d\n", key, ttl))
				if err != nil {
					return fmt.Errorf("failed to write expiry to temporary AOF file: %w", err)
				}
			}
		}
	}

	// Atomically replace the old AOF file with the new one
	oldFilePath := r.aofFilePath + "/aof.txt"
	err = os.Rename(tempFilePath, oldFilePath)
	if err != nil {
		return fmt.Errorf("failed to replace old AOF file: %w", err)
	}

	log.Printf("AOF rewrite completed successfully")
	return nil
}

// formatListForAOF formats a list for writing to the AOF file.
func formatListForAOF(list []interface{}) string {
	var parts []string
	for _, item := range list {
		parts = append(parts, fmt.Sprintf("%v", item))
	}
	return strings.Join(parts, " ")
}

// SaveSnapshot creates a snapshot of the current state of the database.
func (r *Tealis) SaveSnapshot() error {
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
func (r *Tealis) LoadSnapshot() error {
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

func (r *Tealis) GetClientConnection(clientID string) (interface{}, error) {
	r.Mu.RLock()
	defer r.Mu.RUnlock()

	conn, exists := r.ClientConnections[clientID]
	if !exists {
		return nil, fmt.Errorf("client %s not found", clientID)
	}
	return conn, nil
}

func (r *Tealis) StartSnapshotScheduler(ctx context.Context, interval time.Duration) {
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

// APPENDTO appends a command to the transaction queue for the given client.
func (r *Tealis) APPENDTO(clientID string, command string) string {
	r.Mu.Lock()
	defer r.Mu.Unlock()

	// Check if the client has an active transaction
	if _, ok := r.Transactions[clientID]; !ok {
		return "- appendto ERR No transaction started\r\n"
	}

	// Append the command to the client's transaction queue
	r.Transactions[clientID] = append(r.Transactions[clientID], command)

	// Return a success message
	return "+QUEUED\r\n"
}

// TTL returns the time-to-live (TTL) of a key in seconds.
func (r *Tealis) TTL(key string) int64 {
	r.Mu.RLock()
	defer r.Mu.RUnlock()

	// Check if the key exists
	if _, exists := r.Store[key]; !exists {
		return -2 // Key does not exist
	}

	// Check if the key has an expiry
	if expiry, exists := r.Expiries[key]; exists {
		remaining := time.Until(expiry).Seconds()
		if remaining > 0 {
			return int64(remaining) // Return remaining time in seconds
		}
		return -2 // Key has expired
	}

	return -1 // Key exists but has no expiry
}

// PERSIST removes the expiry from a key, making it persistent.
func (r *Tealis) PERSIST(key string) int {
	r.Mu.Lock()
	defer r.Mu.Unlock()

	// Check if the key exists
	if _, exists := r.Store[key]; !exists {
		return 0 // Key does not exist
	}

	// Check if the key has an expiry
	if _, exists := r.Expiries[key]; exists {
		delete(r.Expiries, key) // Remove the expiry
		return 1                // Expiry removed
	}

	return 0 // Key exists but has no expiry
}
