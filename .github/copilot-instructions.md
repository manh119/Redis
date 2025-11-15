# Redis Clone - Copilot Instructions

## Project Overview
A Go-based client-server application using the **thread-per-connection model** for concurrent client handling.

## Technology Stack
- **Language**: Go 1.21+
- **Concurrency Model**: Goroutines (lightweight threads)
- **Synchronization**: sync.RWMutex for thread-safe access
- **Protocol**: TCP sockets
- **Port**: 8080

## Project Status
✅ Project scaffolded and ready to run
✅ Server: Thread-per-connection model implemented
✅ Client: Interactive interface for sending multiple requests
✅ Thread-safe: Mutex protection on shared data
✅ Documentation: Complete README with examples

## Building and Running

### Build
```bash
go build -o server ./server
go build -o client ./client
```

### Run Server
```bash
go run ./server/main.go
```

### Run Client(s)
```bash
go run ./client/main.go
```

Multiple clients can connect simultaneously to test concurrent handling.

## Supported Features
- SET/GET/DELETE/LIST commands
- Multiple concurrent client connections
- Thread-safe key-value store
- Error handling and logging
- Health check (PING command)
