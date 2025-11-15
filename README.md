# Redis Clone - Go Client-Server Application

A simple client-server application written in Go using a **thread-per-connection model** where each client connection is handled in its own goroutine.

## Architecture

### Thread-Per-Connection Model
- **Server**: Listens for incoming TCP connections on port `8080`
- **Each Connection**: Handled by a dedicated goroutine (lightweight thread)
- **Concurrency**: Multiple clients can connect simultaneously
- **Thread Safety**: Shared data (key-value store) is protected with `sync.RWMutex`

## Project Structure

```
redis-clone/
├── server/
│   └── main.go          # Server implementation
├── client/
│   └── main.go          # Client implementation
├── go.mod              # Go module file
└── README.md           # This file
```

## Building the Project

```bash
# Build server
go build -o server ./server

# Build client
go build -o client ./client
```

## Running the Project

### Terminal 1: Start the Server
```bash
go run ./server/main.go
# Output: Server listening on :8080
```

### Terminal 2: Run the Client(s)
```bash
go run ./client/main.go
```

You can open **multiple terminals** and run multiple clients simultaneously!

## Supported Commands

| Command | Example | Description |
|---------|---------|-------------|
| **SET** | `SET key value` | Store a key-value pair |
| **GET** | `GET key` | Retrieve value for a key |
| **DELETE** | `DELETE key` | Remove a key-value pair |
| **LIST** | `LIST` | Show all keys in store |
| **PING** | `PING` | Server health check |
| **QUIT** | `QUIT` | Disconnect from server |

## Example Session

```
Connected to server at localhost:8080
Enter command: SET user1 Alice
Server: OK: Set user1 = Alice
Enter command: SET user2 Bob
Server: OK: Set user2 = Bob
Enter command: GET user1
Server: OK: Alice
Enter command: LIST
Server: OK: Keys: user1, user2
Enter command: DELETE user1
Server: OK: Deleted user1
Enter command: QUIT
Exiting...
```

## Features

### ✅ Thread-Per-Connection Model
- Each client connection runs in its own goroutine
- Non-blocking concurrent handling of multiple clients
- Efficient resource utilization with Go's lightweight goroutines

### ✅ Thread-Safe Operations
- `sync.RWMutex` protects shared key-value store
- Multiple readers can access data simultaneously
- Exclusive access during write operations

### ✅ Multiple Request Support
- Clients can send multiple requests sequentially
- Server processes each request independently
- Full-duplex communication

### ✅ Error Handling
- Invalid commands are caught and reported
- Missing keys are handled gracefully
- Connection errors are logged

## Implementation Details

### Server (`server/main.go`)

1. **Main Loop**: Accepts incoming connections in a loop
2. **Goroutine per Connection**: Each accepted connection spawns a new goroutine
3. **Request Processing**: Reads commands, processes them, and sends responses
4. **Concurrency Control**: Uses `sync.RWMutex` for thread-safe access to the store

```go
for {
    conn, err := listener.Accept()
    go handleConnection(conn)  // Thread-per-connection
}
```

### Client (`client/main.go`)

- Connects to the server
- Provides interactive command prompt
- Sends commands and receives responses
- Supports synchronous and asynchronous request patterns

## Testing

### Test Multiple Concurrent Clients

```bash
# Terminal 1
go run ./server/main.go

# Terminal 2
go run ./client/main.go

# Terminal 3
go run ./client/main.go

# Terminal 4
go run ./client/main.go
```

Each terminal can send commands simultaneously and the server handles them concurrently.

### Performance Notes

- Can handle hundreds of concurrent connections
- Minimal memory overhead per connection (goroutines are lightweight)
- Mutex contention only during store access (< 1ms typically)

## Future Enhancements

- [ ] Persistent storage (file-based or database)
- [ ] Expiration times for keys (TTL)
- [ ] Data types support (lists, sets, hashes)
- [ ] Pub/Sub messaging
- [ ] Replication
- [ ] Cluster support
