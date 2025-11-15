package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

func main() {
	// Connect to the server
	conn, err := net.Dial("tcp", "localhost:8080")
	if err != nil {
		fmt.Println("Error connecting to server:", err)
		return
	}
	defer conn.Close()

	fmt.Println("Connected to server at localhost:8080")
	fmt.Println("You can now send commands: SET, GET, DELETE, LIST, PING, QUIT")
	fmt.Println("Example: SET key value, GET key, DELETE key, LIST, PING")
	fmt.Println()

	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)

	// Option 1: Interactive mode - read from stdin
	interactiveMode(conn, reader, writer)
}

// interactiveMode handles interactive command input from user
func interactiveMode(conn net.Conn, reader *bufio.Reader, writer *bufio.Writer) {
	inputScanner := bufio.NewScanner(conn)
	var wg sync.WaitGroup

	// Goroutine to read responses from server
	wg.Add(1)
	go func() {
		defer wg.Done()
		for inputScanner.Scan() {
			response := inputScanner.Text()
			fmt.Printf("Server: %s\n", response)
			if response == "QUIT" {
				return
			}
		}
	}()

	// Main thread: read user input and send commands
	inputReader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("Enter command: ")
		command, _ := inputReader.ReadString('\n')
		command = strings.TrimSpace(command)

		if command == "" {
			continue
		}

		// Send command to server
		fmt.Fprintf(conn, "%s\n", command)
		conn.SetWriteDeadline(time.Now().Add(5 * time.Second))

		if strings.ToUpper(command) == "QUIT" {
			fmt.Println("Exiting...")
			break
		}
	}

	wg.Wait()
}

// sendMultipleRequests demonstrates sending multiple requests asynchronously
func sendMultipleRequests(conn net.Conn) {
	writer := bufio.NewWriter(conn)
	reader := bufio.NewReader(conn)

	requests := []string{
		"SET user1 Alice",
		"SET user2 Bob",
		"SET user3 Charlie",
		"GET user1",
		"GET user2",
		"LIST",
		"DELETE user2",
		"GET user2",
		"LIST",
		"PING",
	}

	var wg sync.WaitGroup

	// Send requests in parallel
	for i, req := range requests {
		wg.Add(1)
		go func(index int, command string) {
			defer wg.Done()
			fmt.Printf("[Request %d] Sending: %s\n", index+1, command)
			fmt.Fprintf(writer, "%s\n", command)
			writer.Flush()

			// Read response
			response, _ := reader.ReadString('\n')
			fmt.Printf("[Response %d] %s\n", index+1, strings.TrimSpace(response))
		}(i, req)

		// Small delay between sends to avoid race conditions
		time.Sleep(100 * time.Millisecond)
	}

	wg.Wait()
}
