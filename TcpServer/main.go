package main

import (
	"log"
	"net"
	"syscall"
)

// func handleConnection(conn net.Conn) {
// 	// 1. read data from client
// 	var buf []byte = make([]byte, 1000)
// 	_, err := conn.Read(buf)
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	// 2. process
// 	// time.Sleep(time.Second * 10)
// 	//log.Printf("Processed request from %s in thread %d", conn.RemoteAddr(), getThreadID())

//		// 3. reply
//		conn.Write([]byte("+PONG\r\n"))
//		// conn.Close()
//	}
func handleConnection(conn net.Conn) {
	log.Printf("New connection from %s in thread %d", conn.RemoteAddr(), getThreadID())
	defer conn.Close()

	buf := make([]byte, 4096)

	for {
		n, err := conn.Read(buf)
		if err != nil {
			return // client đóng connection
		}

		// ignore actual RESP parsing — just respond PONG
		_ = n
		conn.Write([]byte("+PONG\r\n"))
	}
}

func main() {
	listener, err := net.Listen("tcp", ":3000")
	log.Println("Server is listening on port 3000")

	if err != nil {
		log.Fatal(err)
	}
	for {
		// conn == socket == dedicated communication channel
		conn, err := listener.Accept()
		if err != nil {
			log.Fatal(err)
		}
		// create a new goroutine to handle the connection
		go handleConnection(conn)
	}
}

func getThreadID() int {
	tid, _, _ := syscall.RawSyscall(syscall.SYS_GETTID, 0, 0, 0)
	return int(tid)
}
