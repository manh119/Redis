package main

import (
	"log"
	"net"
)

// test connect TCP : nc localhost 3000
func main() {
	listener, err := net.Listen("tcp", ":3000")
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("server listen at port %v", listener.Addr())

	defer listener.Close()

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Fatal(err)
		}

		// each goroutine handle a connection, not block the main thread
		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	log.Printf("handle connection from remote adrress %v", conn.RemoteAddr())
	defer conn.Close()

	buff := make([]byte, 1024)
	for {
		n, err := conn.Read(buff)

		if err != nil {
			log.Printf("read error: %v", err)
			return
		}

		log.Printf("read %d bytes : %s", n, buff[:n])
		_, err = conn.Write(buff[:n])
		if err != nil {
			return
		}
		log.Printf("write %s", buff[:n])
	}
}
