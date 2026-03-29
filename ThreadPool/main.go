package main

import (
	"log"
	"net"
	"time"
)

/// implement thread pool
/// just add any new connection to the shared-queue
/// a pool of thread (workers) just wait for the new data in the queue to consumer

type ThreadPool struct {
	queue   chan net.Conn // shared-queue worker works on
	workers []Worker
	n       int
}

type Worker struct {
	id    int
	queue chan net.Conn // shared-queue worker works on
}

func NewThreadPool(n int) *ThreadPool {

	threadPool := &ThreadPool{
		queue:   make(chan net.Conn, n),
		workers: make([]Worker, n),
		n:       n,
	}

	for i := 0; i < n; i++ {
		threadPool.workers[i].id = i
		threadPool.workers[i].queue = threadPool.queue
	}

	log.Printf("Created threadpool with %d workers", n)
	return threadPool
}

func (pool ThreadPool) Start() {
	for i := 0; i < pool.n; i++ {
		go pool.workers[i].run()
	}
}

func (worker Worker) run() {
	for conn := range worker.queue {
		log.Printf("connection at address %s are handled by worker %d", conn.RemoteAddr(), worker.id)
		handleConnection(conn)
	}
}

func (pool ThreadPool) addConn(conn net.Conn) {
	pool.queue <- conn
}

func handleConnection(conn net.Conn) {
	defer conn.Close()
	buf := make([]byte, 4096)

	for {
		_, err := conn.Read(buf)
		if err != nil {
			return // client đóng connection
		}
		time.Sleep(100 * time.Millisecond)
		conn.Write([]byte("+PONG\r\n"))
	}
}

func main() {
	listener, err := net.Listen("tcp", ":3001")
	if err != nil {
		log.Fatal(err)
	}
	defer listener.Close()

	threadPool := NewThreadPool(5)
	threadPool.Start()

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Fatal(err)
		}
		//go handleConnection(conn)
		threadPool.addConn(conn)
	}
}
