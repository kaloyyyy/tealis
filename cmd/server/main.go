package main

import (
	"context"
	"github.com/gorilla/websocket"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"tealis/internal/protocol"
	"tealis/internal/storage"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins (adjust for security if needed)
	},
}

func websocketHandler(store *storage.RedisClone, w http.ResponseWriter, r *http.Request) {
	// Upgrade the HTTP connection to a WebSocket connection
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Error upgrading to WebSocket: %v", err)
		return
	}
	defer conn.Close()

	// Handle incoming WebSocket messages
	for {
		// Read message from client
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Printf("WebSocket read error: %v", err)
			break
		}

		// Log the received command
		command := string(message)
		log.Printf("Received WebSocket command: %s", command)

		// Process the command and get the response
		parts := protocol.ParseCommand(command)
		response := storage.ProcessCommand(parts, store, "0")

		// Send the response back to the client
		if err := conn.WriteMessage(websocket.TextMessage, []byte(response)); err != nil {
			log.Printf("WebSocket write error: %v", err)
			break
		}
	}
}
func main() {
	// Create the Redis clone instance
	store := storage.NewRedisClone()
	// Create a context for cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start background cleanup task for expired keys
	store.StartCleanup(ctx)
	go func() {
		// Start WebSocket server on a separate port (e.g., 8080)
		http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
			websocketHandler(store, w, r)
		})
		log.Println("WebSocket server is running on port 8080...")
		log.Fatal(http.ListenAndServe(":8080", nil))
	}()
	// Set up a listener on port 6379
	listener, err := net.Listen("tcp", ":6379")
	if err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
	defer listener.Close()

	// Log that the server is running
	log.Println("Redis clone is running on port 6379...")
	log.Println("use `telnet 127.0.0.1 6379` to connect")
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

	clientID := clientAddr // For simplicity, use the client's address as the ID
	store.Mu.Lock()
	store.ClientConnections[clientID] = conn
	store.Mu.Unlock()
	defer func() {
		log.Printf("Client %s disconnected.", clientAddr)
		store.Mu.Lock()
		delete(store.ClientConnections, clientID)
		store.Mu.Unlock()
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
				response := storage.ProcessCommand(parts, store, clientID)

				// Send the response back to the client
				conn.Write([]byte(response + "\r\n"))
				if strings.ToUpper(parts[0]) == "QUIT" {
					log.Printf("Client %s sent QUIT. Closing connection.", clientAddr)
					return // Break out of the loop to close the connection
				}
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
