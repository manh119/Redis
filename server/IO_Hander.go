package server

import (
	"fmt"
	"hash/fnv"
	"log"
	"math/rand"
	"syscall"

	"github.com/manh119/Redis/internal/core/io_multiplexing"
	"github.com/manh119/Redis/internal/core/resp"
)

type IOHander struct {
	multiplexer []*io_multiplexing.Epoll
	workers     []*Worker
	size        int
}

func (handler *IOHander) addNewConnection(fdListener int) error {
	fdNewCon, connAdrr, err := syscall.Accept(fdListener)
	ip, port := parseSockaddr(connAdrr)
	log.Printf("new connection fd=%d from %s:%d", fdNewCon, ip, port)
	if err != nil {
		log.Printf("error connect to %s with error : %s", connAdrr, err.Error())
		return err
	}
	syscall.SetNonblock(fdNewCon, true)
	// random added
	randIndex := rand.Intn(handler.size)
	err = handler.multiplexer[randIndex].Monitor(io_multiplexing.Event{
		fdNewCon, io_multiplexing.OpRead,
	})
	return err
}

func NewIOHandler(size int, workers []*Worker) *IOHander {
	handler := &IOHander{
		multiplexer: make([]*io_multiplexing.Epoll, size),
		size:        size,
		workers:     workers,
	}

	for i := 0; i < size; i++ {
		ioplx, err := io_multiplexing.CreateIOMultiplexer()
		if err != nil {
			return nil
		}
		handler.multiplexer[i] = ioplx
	}

	handler.run()
	return handler
}

func (handler *IOHander) run() {
	for i := 0; i < handler.size; i++ {
		go func(id int) {
			log.Printf("Started new IO handler id = %d", id)
			for {
				events, err := handler.multiplexer[id].Wait()
				if err != nil {
					log.Printf(err.Error())
					continue
				}

				for j := 0; j < len(events); j++ {
					handler.handleCommand(handler.multiplexer[id], events[j].Fd)
				}
			}
		}(i)
	}
}

func (handler *IOHander) handleCommand(multiplexer *io_multiplexing.Epoll, fd int) {
	buffer := make([]byte, 1024)
	n, err := syscall.Read(fd, buffer)

	// n == 0 nghĩa là client đóng connection
	if n == 0 {
		multiplexer.Close()
		err := syscall.Close(fd)
		if err != nil {
			log.Printf(err.Error())
			return
		}
		return
	}

	if err != nil {
		// chưa có data → bỏ qua
		if err == syscall.EAGAIN || err == syscall.EWOULDBLOCK {
			return
		}
		syscall.Close(fd)
		return
	}

	// get key from command
	key, done := getKey(err, buffer, n)
	if done {
		log.Printf("failed to get key")
		return
	}

	hasher := fnv.New32a()
	hasher.Write([]byte(key))
	workerID := int(hasher.Sum32()) % len(handler.workers)
	request := buffer[:n]

	handler.workers[workerID].taskChan <- request
	response := <-handler.workers[workerID].replyChan

	syscall.Write(fd, response)
}

func getKey(err error, buffer []byte, n int) (string, bool) {
	decodeRequest, _, err := resp.ReadArray(buffer[:n])
	if err != nil {
		log.Printf("Error decode")
		return "", true
	}
	log.Printf("decodedMess: %s", decodeRequest)
	arr, ok := decodeRequest.([]any)
	if !ok || decodeRequest == nil || len(arr) == 0 {
		log.Printf("Error decode")
	}
	var args []string
	for i := 1; i < len(arr); i++ {
		args = append(args, fmt.Sprintf("%v", arr[i]))
	}
	key := ""
	if len(arr) >= 2 {
		key = args[0]
		return key, false
	}
	return key, true
}
