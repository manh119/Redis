package main

import (
	"log"
	"net"
	"syscall"
)

// element in the queue
type Job struct {
	conn net.Conn
}

// represent the thread in the pool
type Worker struct {
	id      int
	jobChan chan Job
}

// represent the thread pool
type Pool struct {
	jobQueue chan Job
	workers  []*Worker
}

// create a new worker
func NewWorker(id int, jobChan chan Job) *Worker {
	return &Worker{
		id:      id,
		jobChan: jobChan,
	}
}

func (w *Worker) Start() {
	go func() {
		for job := range w.jobChan {
			log.Printf("Worker %d is handling job from %s", w.id, job.conn.RemoteAddr())
			handleConnection(job.conn)
			log.Printf("Worker %d finished job from %s", w.id, job.conn.RemoteAddr())
		}
	}()
}

func NewPool(numOfWorker int) *Pool {
	return &Pool{
		jobQueue: make(chan Job),
		workers:  make([]*Worker, numOfWorker),
	}
}

// push job to queue
func (p *Pool) AddJob(conn net.Conn) {
	p.jobQueue <- Job{conn: conn}
}

func (p *Pool) Start() {
	log.Printf("Starting thread pool with %d workers", len(p.workers))
	for i := 0; i < len(p.workers); i++ {
		log.Printf("Starting worker %d", i)
		worker := NewWorker(i, p.jobQueue)
		p.workers[i] = worker
		worker.Start()
	}
}

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
	listener, err := net.Listen("tcp", ":3001")
	if err != nil {
		log.Fatal(err)
	}
	defer listener.Close()

	// 1 pool with 2 threads
	pool := NewPool(200)
	pool.Start()

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Fatal(err)
		}
		//go handleConnection(conn)
		pool.AddJob(conn)
	}
}

func getThreadID() int {
	tid, _, _ := syscall.RawSyscall(syscall.SYS_GETTID, 0, 0, 0)
	return int(tid)
}
