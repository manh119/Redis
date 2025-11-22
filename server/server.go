package server

import (
	"io"
	"log"
	"net"
	"redis-clone/internal/core/io_multiplexing"
	"syscall"
)

func readCommand(fd int) (string, error) {
	var buf = make([]byte, 512)
	n, err := syscall.Read(fd, buf)
	if err != nil {
		return "", err
	}
	if n == 0 {
		return "", io.EOF
	}
	return string(buf[:n]), nil
}

func respond(data string, fd int) error {
	if _, err := syscall.Write(fd, []byte(data)); err != nil {
		return err
	}
	return nil
}

func RunIoMultiplexingServer() {
	// 1. create io multiplexer and list on port 4000
	log.Println("Starting IO Multiplexing Server on port 4000")
	listener, err := net.Listen("tcp", ":4000")
	if err != nil {
		log.Fatal(err)
	}
	defer listener.Close()

	// 2. Create epoll instance
	ioMultiplexer, err := io_multiplexing.CreateIOMultiplexer()
	if err != nil {
		log.Fatal(err)
	}
	defer ioMultiplexer.Close()

	// 3. Monitor port 4000 for new connections
	listenerFile, err := listener.(*net.TCPListener).File()
	if err != nil {
		log.Fatal(err)
	}
	defer listenerFile.Close()

	listenerEvent := io_multiplexing.Event{
		Fd: int(listenerFile.Fd()),
		Op: io_multiplexing.OpRead,
	}
	if err := ioMultiplexer.Monitor(listenerEvent); err != nil {
		log.Fatal(err)
	}

	// 4. Map to store active connections
	connections := make(map[int]net.Conn)

	// 5. Event loop
	for {
		// wait for events -- TODO : blocking hay non-blocking ??
		events, err := ioMultiplexer.Wait()
		if err != nil {
			log.Fatal(err)
		}
		for _, event := range events {
			// New connection
			if event.Fd == int(listenerFile.Fd()) {

				// Accept new connection - ?? TODO ? accept o day nghia la gi
				conn, err := listener.Accept()
				if err != nil {
					log.Fatal(err)
				}
				log.Printf("Accepted new connection from %s", conn.RemoteAddr())

				connFile, err := conn.(*net.TCPConn).File()
				if err != nil {
					log.Fatal(err)
				}

				connEvent := io_multiplexing.Event{
					Fd: int(connFile.Fd()),
					Op: io_multiplexing.OpRead,
				}
				if err := ioMultiplexer.Monitor(connEvent); err != nil {
					log.Fatal(err)
				}
				connections[int(connFile.Fd())] = conn
			} else {
				// Existing connection has data to read
				conn := connections[event.Fd]
				command, err := readCommand(event.Fd)
				if err != nil {
					if err == io.EOF {
						log.Printf("Connection closed by client %s", conn.RemoteAddr())
						conn.Close()
						delete(connections, event.Fd)
						continue
					}
					log.Fatal(err)
				}
				log.Printf("Received command from %s: %s", conn.RemoteAddr(), command)

				// Simple RESP parsing (only handles PING)
				var response = "hello world"

				if err := respond(response, event.Fd); err != nil {
					log.Fatal(err)
				}
			}
		}
	}

}
