package server

import (
	"errors"
	"log"
	"net"
	"strconv"
	"syscall"

	"github.com/manh119/Redis/internal/core"
	"github.com/manh119/Redis/internal/core/data_structure"
	"github.com/manh119/Redis/internal/core/resp"
)

var dictStore = data_structure.NewDictionary()

func RunIoMultiplexingServer() {
	listener, err := net.Listen("tcp", ":4000")
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
	fdEpoll, err := syscall.EpollCreate1(0)
	if err != nil {
		log.Fatal(err)
	}
	defer syscall.Close(fdEpoll)

	// 2. add tcp connection to the epoll to handle new connection
	listenerEvent := &syscall.EpollEvent{Events: syscall.EPOLLIN, Fd: int32(fdListener)}
	err = syscall.EpollCtl(fdEpoll, syscall.EPOLL_CTL_ADD, fdListener, listenerEvent)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Server đang lắng nghe trên port :4000...")

	// 3. wait for new event in the epoll, event loop
	bufferEvents := make([]syscall.EpollEvent, 100)
	for {
		n, err := syscall.EpollWait(fdEpoll, bufferEvents, -1)
		if err != nil {
			log.Printf(err.Error())
			continue
		}

		for i := 0; i < n; i++ {
			currentFd := bufferEvents[i].Fd
			if currentFd == int32(fdListener) {
				fdConn, connAdrr, err := syscall.Accept(fdListener)
				ip, port := parseSockaddr(connAdrr)
				log.Printf("new connection fd=%d from %s:%d", fdConn, ip, port)
				if err != nil {
					log.Printf("error connect to %s with error : %s", connAdrr, err.Error())
					continue
				}
				syscall.SetNonblock(fdConn, true)

				// 4. add new connection to the epoll to monitor
				connEvent := &syscall.EpollEvent{Events: syscall.EPOLLIN, Fd: int32(fdConn)}
				err = syscall.EpollCtl(fdEpoll, syscall.EPOLL_CTL_ADD, fdConn, connEvent)
				if err != nil {
					log.Printf(err.Error())
				}
			} else {
				log.Printf("new change from fd : %d", currentFd)
				readCommandAndReponse(fdEpoll, int(currentFd))
			}
		}
	}
}

func readCommandAndReponse(fdEpoll int, fd int) {
	buffer := make([]byte, 1024)
	n, err := syscall.Read(fd, buffer)

	// n == 0 nghĩa là client đóng connection
	if n == 0 {
		syscall.Close(fd)
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
	// 1. decode request
	decodeRequest, _, err := resp.ReadArray(buffer[:n])
	if err != nil {
		log.Printf("Error decode")
		return
	}
	log.Printf("decodeMess: %s", decodeRequest)

	// 2. read command
	response, err := handleCommand(decodeRequest)
	if err != nil {
		response = err.Error()
	}

	// 3. encode response
	encodedRes, err := resp.Encode(response)

	// 4. response
	syscall.Write(fd, []byte(encodedRes))
}

func handleCommand(decodeRequest any) (any, error) {
	arr, ok := decodeRequest.([]any)
	if !ok {
		return "", errors.New("invalid command")
	}
	if decodeRequest == nil || len(arr) == 0 {
		return "", errors.New("invalid command")
	}

	cmd := core.Command{Cmd: arr[0].(string), Args: arr[1:]}

	switch cmd.Cmd {
	case "ping":
		return handlePing(cmd)
	case "get":
		return handleGet(cmd)
	case "set":
		return handleSet(cmd)
	case "ttl":
		return handleTTL(cmd)
	}
	return "", nil
}

func handleTTL(cmd core.Command) (any, error) {
	if len(cmd.Args) == 1 {
		key, ok := cmd.Args[0].(string)
		if !ok {
			return "", errors.New("ERR value is not a valid string")
		}
		return dictStore.Ttl(key), nil
	}
	return "", errors.New("invalid command")
}

func handlePing(cmd core.Command) (string, error) {
	if cmd.Args == nil || len(cmd.Args) == 0 {
		return "pong", nil
	}

	if len(cmd.Args) == 1 {
		return cmd.Args[0].(string), nil
	}

	return "", errors.New("invalid command")
}

func handleGet(cmd core.Command) (any, error) {
	if len(cmd.Args) == 1 {
		key, ok := cmd.Args[0].(string)
		if !ok {
			return "", errors.New("ERR value is not a valid string")
		}
		return dictStore.Get(key), nil
	}
	return "", errors.New("invalid command")
}

func handleSet(cmd core.Command) (string, error) {
	argCount := len(cmd.Args)
	if argCount != 2 && argCount != 4 {
		return "", errors.New("ERR wrong number of arguments for 'set' command")
	}
	key, ok1 := cmd.Args[0].(string)
	if !ok1 {
		return "", errors.New("ERR key must be a string")
	}
	value := cmd.Args[1]
	var ttl int64 = -1
	if argCount == 4 {
		ttlStr, ok := cmd.Args[3].(string)
		if !ok {
			return "", errors.New("ERR value is not an integer or out of range")
		}
		parsedTTL, err := strconv.ParseInt(ttlStr, 10, 64)
		if err != nil {
			return "", errors.New("ERR value is not an integer or out of range")
		}
		ttl = parsedTTL
	}
	dictStore.Set(key, value, ttl*1000)
	return "OK", nil
}

func parseSockaddr(addr syscall.Sockaddr) (ip string, port int) {
	switch a := addr.(type) {
	case *syscall.SockaddrInet4:
		ip = net.IP(a.Addr[:]).String()
		port = a.Port
	case *syscall.SockaddrInet6:
		ip = net.IP(a.Addr[:]).String()
		port = a.Port
	default:
		ip = "unknown"
		port = 0
	}
	return
}
