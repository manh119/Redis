package main

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"sync"
)

var (
	// Thread-safe map to store key-value pairs
	store = make(map[string]string)
	mu    sync.RWMutex
)

func main() {
	// Listen on port 8080
	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		fmt.Println("Error listening:", err)
		return
	}
	defer listener.Close()

	fmt.Println("Server listening on :8080")
	fmt.Println("Thread-per-connection model active")

	for {
		// Accept incoming connections
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting connection:", err)
			continue
		}

		// Handle each connection in a separate goroutine (thread-per-connection)
		go handleConnection(conn)
	}
}

// handleConnection processes requests from a single client
// This goroutine runs in its own thread for the lifetime of the connection
func handleConnection(conn net.Conn) {
	defer conn.Close()

	remoteAddr := conn.RemoteAddr().String()
	fmt.Printf("[%s] Client connected\n", remoteAddr)

	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)

	for {
		// Read command from client
		line, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf("[%s] Client disconnected\n", remoteAddr)
			break
		}

		// Process the command
		command := strings.TrimSpace(line)
		if command == "" {
			continue
		}

		response := processCommand(command)

		// Send response back to client
		fmt.Fprintf(writer, "%s\n", response)
		writer.Flush()
	}
}

// processCommand handles different commands: SET, GET, DELETE, LIST
func processCommand(command string) string {
	parts := strings.Fields(command)

	if len(parts) == 0 {
		return "ERROR: Empty command"
	}

	cmd := strings.ToUpper(parts[0])

	switch cmd {
	case "SET":
		if len(parts) < 3 {
			return "ERROR: SET requires key and value"
		}
		key := parts[1]
		value := strings.Join(parts[2:], " ")
		mu.Lock()
		store[key] = value
		mu.Unlock()
		return fmt.Sprintf("OK: Set %s = %s", key, value)

	case "GET":
		if len(parts) < 2 {
			return "ERROR: GET requires key"
		}
		key := parts[1]
		mu.RLock()
		value, exists := store[key]
		mu.RUnlock()

		if !exists {
			return fmt.Sprintf("ERROR: Key '%s' not found", key)
		}
		return fmt.Sprintf("OK: %s", value)

	case "DELETE":
		if len(parts) < 2 {
			return "ERROR: DELETE requires key"
		}
		key := parts[1]
		mu.Lock()
		_, exists := store[key]
		delete(store, key)
		mu.Unlock()

		if !exists {
			return fmt.Sprintf("ERROR: Key '%s' not found", key)
		}
		return fmt.Sprintf("OK: Deleted %s", key)

	case "LIST":
		mu.RLock()
		defer mu.RUnlock()
		if len(store) == 0 {
			return "OK: Store is empty"
		}
		var result strings.Builder
		result.WriteString("OK: Keys: ")
		i := 0
		for key := range store {
			if i > 0 {
				result.WriteString(", ")
			}
			result.WriteString(key)
			i++
		}
		return result.String()

	case "PING":
		return "PONG"

	case "QUIT":
		return "QUIT"

	default:
		return fmt.Sprintf("ERROR: Unknown command '%s'", cmd)
	}
}
