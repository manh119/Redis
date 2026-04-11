package server

import (
	"errors"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"

	"github.com/manh119/Redis/internal/config"
	storage "github.com/manh119/Redis/internal/core"
	"github.com/manh119/Redis/internal/core/command"
	"github.com/manh119/Redis/internal/core/data_structure"
	"github.com/manh119/Redis/internal/core/io_multiplexing"
	"github.com/manh119/Redis/internal/core/resp"
)

type Workers struct {
	workers []*Worker
	size    int
}

type Worker struct {
	id        int
	taskChan  chan []byte
	replyChan chan []byte
	dictStore *data_structure.Dictionary
}

func NewListWorkers(size int) *Workers {
	workers := &Workers{
		workers: make([]*Worker, size),
		size:    size,
	}
	for i := 0; i < size; i++ {
		workers.workers[i] = &Worker{
			id:        i,
			taskChan:  make(chan []byte, 1024),
			replyChan: make(chan []byte, 1024),
			dictStore: data_structure.NewDictionary(),
		}
	}
	workers.run()
	return workers
}

func (wk *Workers) run() {
	for i := 0; i < wk.size; i++ {
		go func(index int) {
			log.Printf("worker %d starting", index)
			for {
				request := <-wk.workers[index].taskChan

				decodeRequest, _, err := resp.ReadArray(request)
				if err != nil {
					log.Printf("Error decode")
					continue
				}
				//log.Printf("decodedMess: %s", decodeRequest)

				// 2. read command
				response, err := wk.workers[index].HandleCommand(decodeRequest)

				if err != nil {
					log.Printf(err.Error())
					if err.Error() == config.NILL {
						wk.workers[index].replyChan <- []byte(config.NILL)
						continue
					}
					resError, err := resp.Encode(err)
					if err != nil {
						log.Printf("Error encode %s", err.Error())
						continue
					}
					wk.workers[index].replyChan <- []byte(resError)
					continue
				}

				// 3. encode response
				encodedRes, err := resp.Encode(response)
				if err != nil {
					log.Printf("Error encode %s", err.Error())
					continue
				}

				// 4. response
				//log.Printf("encodeMess: %s", encodedRes)
				wk.workers[index].replyChan <- []byte(encodedRes)
			}
		}(i)
	}
}

func RunIoMultiplexingServerMultipleIOHanlder(wg *sync.WaitGroup) {
	defer wg.Done()

	listener, err := net.Listen(config.Protocol, config.Port)
	if err != nil {
		log.Fatal(err)
	}
	defer listener.Close()

	file, err := listener.(*net.TCPListener).File()
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	fdListener := int(file.Fd())
	syscall.SetNonblock(fdListener, true)

	// 1. create epoll instance
	multiplexer, err := io_multiplexing.CreateIOMultiplexer()
	defer multiplexer.Close()

	// 2. add tcp connection to the epoll to handle new connection
	err = multiplexer.Monitor(io_multiplexing.Event{
		fdListener, io_multiplexing.OpRead,
	})
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Server is listening on port%s ...", config.Port)

	// 3. start list of IO handler
	workers := NewListWorkers(3)
	IOHandler := NewIOHandler(3, workers.workers)

	// 4. wait for new event in the epoll, event loop
	for {
		if atomic.LoadInt32(&ServerStatus) == config.ServerStatusShuttingDown {
			handleCleanup(multiplexer, listener)
			return
		}

		events, err := multiplexer.Wait()
		if err != nil {
			log.Printf(err.Error())
			continue
		}

		for i := 0; i < len(events); i++ {
			currentFd := events[i].Fd
			if currentFd == fdListener {
				err = IOHandler.addNewConnection(currentFd)
				if err != nil {
					log.Printf(err.Error())
					continue
				}
			} else {
				log.Fatal("error multipler new connections")
			}
		}
	}
}

func (worker *Worker) HandleCommand(decodeRequest any) (any, error) {
	arr, ok := decodeRequest.([]any)
	if !ok || decodeRequest == nil || len(arr) == 0 {
		return "", errors.New("invalid command")
	}

	cmdName, cmd := convertToCommand(arr)

	switch cmdName {
	case "GET":
		return worker.HandleGet(cmd)
	case "SET":
		return worker.HandleSet(cmd)
	default:
		return "", errors.New("invalid command")
	}
}

func (worker *Worker) HandleGet(cmd *command.Command) (string, error) {
	if len(cmd.Args) == 1 {
		key := cmd.Args[0]
		value, err := worker.dictStore.Get(key)
		if err != nil {
			return "", err
		}
		return value, nil
	}
	return "", errors.New("invalid command")
}

// set key value
// set key value EX ttl
func (worker *Worker) HandleSet(cmd *command.Command) (string, error) {
	argCount := len(cmd.Args)
	if argCount != 2 && argCount != 4 {
		return "", errors.New("ERR wrong number of arguments for 'set' command")
	}
	key := cmd.Args[0]
	value := cmd.Args[1]
	var ttl int64 = -1
	if argCount == 4 {
		ttlStr := cmd.Args[3]
		parsedTTL, err := strconv.ParseInt(ttlStr, 10, 64)
		if err != nil {
			return "", errors.New("ERR value is not an integer or out of range")
		}
		ttl = parsedTTL
		if strings.ToUpper(cmd.Args[2]) == "EX" {
			storage.DictStore.Set(key, value, ttl*1000)
		} else if strings.ToUpper(cmd.Args[2]) == "PX" { // ttl in miliSecond
			storage.DictStore.Set(key, value, ttl)
		} else {
			return "", errors.New("ERR unknown command")
		}
	}
	worker.dictStore.Set(key, value, ttl)
	return "OK", nil
}

func convertToCommand(arr []any) (string, *command.Command) {
	cmdName := arr[0].(string)
	cmdName = strings.ToUpper(cmdName)
	var args []string
	for i := 1; i < len(arr); i++ {
		args = append(args, fmt.Sprintf("%v", arr[i]))
	}

	cmd := command.NewCommand(cmdName, args)
	return cmdName, cmd
}
