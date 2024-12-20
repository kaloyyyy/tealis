package storage

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"tealis/internal/storage"
	"testing"
	"time"
)

func TestAOF(t *testing.T) {
	// Setup
	aofFilePath := "./snapshot"
	snapshotPath := "./snapshot"
	// Clean up the ./snapshot folder and files before starting the test
	if err := os.RemoveAll("./snapshot"); err != nil {
		t.Fatalf("Failed to clean up ./snapshot directory: %v", err)
	}

	// Initialize a Tealis instance with AOF enabled
	r := storage.NewTealis(aofFilePath, snapshotPath, true)
	print(r)
	// Perform operations that modify the database
	r.Set("key1", "value1", 0)
	r.Set("key2", "value2", 0)
	r.EX("key1", 100*time.Second)
	r.Del("key2")
	r.AofFile.Close() // Ensure all data is flushed to the file
	err := r.SaveSnapshot()
	if err != nil {
		t.Errorf("Error saving aof file: %v", err)
	} // Ensure the operations are logged in the AOF file

	// Create a new Tealis instance and reload the AOF file
	r2 := storage.NewTealis(aofFilePath, snapshotPath, true)
	err = r2.LoadSnapshot()
	if err != nil {
		t.Errorf("Error loading aof file: %v", err)
	}
	// Verify the state of the new instance
	if v, ok := r2.Store["key1"].(string); !ok || v != "value1" {
		t.Errorf("Expected key1 to have value 'value1', got '%v'", v)
	}

	// Verify that key2 does not exist
	if _, exists := r2.Store["key2"]; exists {
		t.Errorf("Expected key2 to be deleted")
	}
	time.Sleep(1 * time.Second)
	// Verify expiry times (optional)
	expiry, exists := r2.Expiries["key1"]
	timeNow := time.Now()
	if !exists {
		t.Errorf("Expected key1 to have an expiry")
	} else if expiry.Before(timeNow) {
		t.Errorf("Expected key1 expiry to be in the future, got %v vs now: %v", expiry, timeNow)
		t.Errorf("Time difference: %v", timeNow.Sub(expiry))
	}

	// Append a command to the AOF file
	command := "SET key value"
	r.AppendToAOF(command)

	// Open the AOF file and check if the command was written
	file, err := os.Open(aofFilePath + "/aof.txt")
	if err != nil {
		t.Fatalf("Failed to open AOF file: %v", err)
	}
	fmt.Printf("this file %v", file)

	scanner := bufio.NewScanner(file)
	var found bool
	for scanner.Scan() {
		if strings.TrimSpace(scanner.Text()) == command {
			found = true
			break
		}
	}

	if err := scanner.Err(); err != nil {
		t.Errorf("Error reading AOF file: %v", err)
	}
	defer file.Close()
	if !found {
		t.Errorf("Expected command '%s' not found in AOF file", command)
	}
}
