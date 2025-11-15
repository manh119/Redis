package main

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"sync"
	"sync/atomic"
)

var (
	// Thread-safe map to store key-value pairs
	store = make(map[string]string)
	mu    sync.RWMutex

	// Connection pool statistics
	totalConnections  int64
	activeConnections int64
)

// ConnectionPool manages a pool of worker goroutines
type ConnectionPool struct {
	maxWorkers int
	jobQueue   chan net.Conn
	wg         sync.WaitGroup
	done       chan struct{}
}

// NewConnectionPool creates a new connection pool with specified worker count
func NewConnectionPool(maxWorkers int) *ConnectionPool {
	return &ConnectionPool{
		maxWorkers: maxWorkers,
		jobQueue:   make(chan net.Conn, maxWorkers*2), // buffered channel
		done:       make(chan struct{}),
	}
}

// Start initializes worker goroutines
func (cp *ConnectionPool) Start() {
	for i := 0; i < cp.maxWorkers; i++ {
		cp.wg.Add(1)
		go cp.worker(i)
	}
	fmt.Printf("🚀 Thread pool started with %d workers\n", cp.maxWorkers)
}

// worker processes incoming connections
func (cp *ConnectionPool) worker(id int) {
	defer cp.wg.Done()
	fmt.Printf("   ├─ Worker %d initialized\n", id)

	for {
		select {
		case conn := <-cp.jobQueue:
			if conn == nil {
				fmt.Printf("   ├─ Worker %d shutting down\n", id)
				return
			}
			atomic.AddInt64(&activeConnections, 1)
			fmt.Printf("   ├─ Worker %d handling %s (Active: %d)\n", id, conn.RemoteAddr(), activeConnections)
			handleConnection(conn)
			atomic.AddInt64(&activeConnections, -1)

		case <-cp.done:
			fmt.Printf("   ├─ Worker %d received shutdown signal\n", id)
			return
		}
	}
}

// Submit adds a new connection to the job queue
func (cp *ConnectionPool) Submit(conn net.Conn) error {
	select {
	case cp.jobQueue <- conn:
		atomic.AddInt64(&totalConnections, 1)
		return nil
	case <-cp.done:
		return fmt.Errorf("pool is shutting down")
	}
}

// Shutdown gracefully shuts down the pool
func (cp *ConnectionPool) Shutdown() {
	fmt.Println("\n🛑 Shutting down connection pool...")
	close(cp.done)

	// Close job queue
	close(cp.jobQueue)

	// Wait for all workers to finish
	cp.wg.Wait()
	fmt.Printf("✅ Connection pool shut down complete (Total connections handled: %d)\n", totalConnections)
}

// GetStats returns pool statistics
func GetStats() (int64, int64) {
	return atomic.LoadInt64(&totalConnections), atomic.LoadInt64(&activeConnections)
}

func main() {
	// Create connection pool with 5 worker goroutines
	numWorkers := 2
	pool := NewConnectionPool(numWorkers)
	pool.Start()

	// Listen on port 8080
	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		fmt.Println("Error listening:", err)
		return
	}
	defer listener.Close()

	fmt.Println("Server listening on :8080")
	fmt.Printf("Thread pool model active with %d workers\n", numWorkers)

	// Accept connections
	for {
		// Accept incoming connections
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting connection:", err)
			continue
		}

		// Submit connection to pool
		if err := pool.Submit(conn); err != nil {
			fmt.Printf("Error submitting connection: %v\n", err)
			conn.Close()
		}
	}
}

// handleConnection processes requests from a single client
// This function is called by a worker goroutine from the pool
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

		// Handle QUIT command
		if strings.ToUpper(command) == "QUIT" {
			break
		}
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
