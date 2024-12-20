package main

import (
	"context"
	"fmt"
	"github.com/gorilla/websocket"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"tealis/internal/protocol"
	"tealis/internal/storage"
	"time"
)

func main() {
	// Create the Redis clone instance
	aofFilePath := "./snapshot"
	snapshotPath := "./snapshot"
	store := storage.NewTealis(aofFilePath, snapshotPath, true)
	// Create a context for cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start background cleanup task for expired keys
	store.StartCleanup(ctx)
	store.StartSnapshotScheduler(ctx, 5*60*time.Second)
	// Start WebSocket server on a separate goroutine (e.g., 8080)
	go func() {
		http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
			websocketHandler(store, w, r)
		})
		log.Println("WebSocket server is running on port 8080...")
		log.Fatal(http.ListenAndServe(":8080", nil))
	}()
	// Start HTTP command API server
	go func() {
		http.Handle("/command", handleCommand(store))
		log.Println("HTTP command API server is running on port 8081...")
		log.Fatal(http.ListenAndServe(":8081", nil))
	}()
	// Start HTTP server for the frontend on port 8081
	go serveFrontend()

	// Set up a listener on port 6379 for the Redis clone
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

	// Start accepting connections for Redis clone
	go acceptConnections(listener, store)

	// Block until we receive a shutdown signal
	<-stopChan
	log.Println("Shutting down server...")
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins (adjust for security if needed)
	},
}

func websocketHandler(store *storage.Tealis, w http.ResponseWriter, r *http.Request) {
	// Upgrade the HTTP connection to a WebSocket connection
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Error upgrading to WebSocket: %v", err)
		return
	}
	defer conn.Close()

	// Get the client ID (address of the WebSocket connection)
	clientID := conn.RemoteAddr().String()

	// Add the WebSocket client to ClientConnections map
	store.Mu.Lock()
	store.ClientConnections[clientID] = conn
	store.Mu.Unlock()

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
		log.Printf("Received WebSocket command from %s: %s", clientID, command)

		// Process the command and get the response
		parts := protocol.ParseCommand(command)
		response := storage.ProcessCommand(parts, store, clientID)

		// Send the response back to the client
		if err := conn.WriteMessage(websocket.TextMessage, []byte(response)); err != nil {
			log.Printf("WebSocket write error: %v", err)
			break
		}
	}

	// Clean up on client disconnect
	store.Mu.Lock()
	delete(store.ClientConnections, clientID)
	store.Mu.Unlock()
}

func serveFrontend() {
	// Create a file server to serve static files from the "public" directory
	fileServer := http.FileServer(http.Dir("./public"))

	// Add logging for incoming requests and serve the files
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Serving request: %s", r.URL.Path)
		fileServer.ServeHTTP(w, r)
	})

	log.Println("Frontend server is running on port 8000...")
	log.Fatal(http.ListenAndServe(":8000", nil))
}

func acceptConnections(listener net.Listener, store *storage.Tealis) {
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

func handleConnectionWithRead(conn net.Conn, store *storage.Tealis, clientAddr string) {
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

// HTTP handler for processing raw Redis commands
func handleCommand(store *storage.Tealis) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Only POST method is supported", http.StatusMethodNotAllowed)
			return
		}

		// Read the raw command string from the request body
		body, err := ioutil.ReadAll(r.Body)
		if err != nil || len(body) == 0 {
			http.Error(w, "Invalid or empty command", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		// Parse the command string into parts
		commandString := strings.TrimSpace(string(body))
		if commandString == "" {
			http.Error(w, "Empty command received", http.StatusBadRequest)
			return
		}

		parts := protocol.ParseCommand(commandString)
		if len(parts) == 0 {
			http.Error(w, "Malformed command", http.StatusBadRequest)
			return
		}

		// Process the command using ProcessCommand
		clientID := "HTTP_CLIENT" // Use a generic client ID for HTTP requests
		response := storage.ProcessCommand(parts, store, clientID)

		// Send the response back to the client
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, response)
	}
}
