package main

import (
	"log"
	"net"
	"sync"
	"time"
)

/// implement thread pool
/// just add any new connection to the shared-queue
/// a pool of thread (workers) just wait for the new data in the queue to consumer

type ThreadPool struct {
	queue chan net.Conn // shared-queue worker works on
	wg    sync.WaitGroup
	n     int
}

func NewThreadPool(n int) *ThreadPool {
	log.Printf("Created threadpool with %d workers", n)
	return &ThreadPool{
		queue: make(chan net.Conn, 2*n), // size queue = 2 * number of worker
		wg:    sync.WaitGroup{},
		n:     n,
	}
}

func (pool *ThreadPool) Start() {
	for i := 0; i < pool.n; i++ {
		pool.wg.Add(1)
		go func(workerId int) {
			defer pool.wg.Done()
			for conn := range pool.queue {
				log.Printf("connection from %s are handle by worker %d", conn.RemoteAddr(), workerId)
				handleConnection(conn)
			}
		}(i)
	}
}

// grace full shutdown
func (pool *ThreadPool) Stop() {
	close(pool.queue)
	pool.wg.Wait()
}

func (pool *ThreadPool) addConn(conn net.Conn) {
	pool.queue <- conn
}

func handleConnection(conn net.Conn) {
	defer conn.Close()
	buf := make([]byte, 4096)
	conn.SetReadDeadline(time.Now().Add(time.Second * 5))

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
			log.Printf("Error accepting connection: %s", err)
			continue // continue handle new connection instead of crash main
		}
		threadPool.addConn(conn)
	}

	listener.Close()
	threadPool.Stop()
}
