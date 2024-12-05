package main

import (
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"tealis/internal/commands"
	"tealis/internal/protocol"
	"tealis/internal/storage"
)

func main() {
	// Create the Redis clone instance
	store := storage.NewRedisClone()

	// Start background cleanup task for expired keys
	store.StartCleanup()

	// Set up a listener on port 6379
	listener, err := net.Listen("tcp", ":6379")
	if err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
	defer listener.Close()

	// Log that the server is running
	log.Println("Redis clone is running on port 6379...")

	// Graceful shutdown setup
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, syscall.SIGINT, syscall.SIGTERM)

	// Start accepting connections in a separate goroutine
	go acceptConnections(listener, store)

	// Block until we receive a shutdown signal
	<-stopChan
	log.Println("Shutting down server...")
}

func acceptConnections(listener net.Listener, store *storage.RedisClone) {
	// Continuously accept new client connections
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Error accepting connection: %v", err)
			continue
		}

		// Log new client connection
		clientAddr := conn.RemoteAddr().String()
		log.Printf("New client connected: %s", clientAddr)

		// Handle each connection in a separate goroutine
		go handleConnectionWithRead(conn, store, clientAddr)
	}
}

func handleConnectionWithRead(conn net.Conn, store *storage.RedisClone, clientAddr string) {
	// Create a buffer to read data from the connection
	buf := make([]byte, 1024) // buffer to hold data from client
	var input []byte          // the full command input from the client

	defer func() {
		// Log when the client disconnects
		log.Printf("Client %s disconnected.", clientAddr)
		conn.Close()
	}()
	// Display the prompt for the user
	conn.Write([]byte("> "))
	for {

		// Read data from the connection
		n, err := conn.Read(buf)
		if err != nil {
			// Handle EOF or any read error
			if err.Error() == "EOF" {
				log.Printf("Client %s closed the connection.", clientAddr)
				return
			}
			log.Printf("Error reading input from client %s: %v", clientAddr, err)
			continue
		}
		// Handle backspace (ASCII value 8)
		if buf[0] == 8 { // Backspace ASCII value
			if len(input) > 0 {
				// Remove the last character from input
				//input = input[:len(input)-0]
				//Optionally, send backspace to the client to delete the last character on their screen
				conn.Write([]byte(" \b \b"))

			}
		}
		// Append the read data to the input buffer
		input = append(input, buf[:n]...)

		// Check if we have reached the end of the command (i.e., newline character)
		if strings.Contains(string(input), "\n") {
			// Trim any extra spaces or newline characters from the input
			line := strings.TrimSpace(string(cleanBytes(input)))

			// Log the received command from the client
			log.Printf("Received command from %s: %s", clientAddr, line)

			// Parse the command and process it
			parts := protocol.ParseCommand(line)

			if len(parts) > 0 {
				// Process the command and get the response
				response := commands.ProcessCommand(parts, store)

				// Send the response back to the client
				conn.Write([]byte(response + "\r\n"))
			}
			// Clear the input buffer after processing the command
			input = nil
		}
	}
}

func cleanBytes(data []byte) []byte {
	var result []byte
	for _, b := range data {
		if b == '\b' { // Check for backspace character
			if len(result) > 0 {
				result = result[:len(result)-1] // Remove the last byte if present
			}
		} else {
			result = append(result, b) // Append the current byte
		}
	}
	return result
}
