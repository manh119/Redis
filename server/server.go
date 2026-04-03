package server

import (
	"errors"
	"fmt"
	"log"
	"net"
	"strings"
	"syscall"

	"github.com/manh119/Redis/internal/core"
	"github.com/manh119/Redis/internal/core/resp"
)

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
		connEvent := &syscall.EpollEvent{Events: syscall.EPOLLIN, Fd: int32(fd)}
		err = syscall.EpollCtl(fdEpoll, syscall.EPOLL_CTL_DEL, fd, connEvent)
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
		log.Printf(err.Error())
		response = err
	}

	// 3. encode response
	encodedRes, err := resp.Encode(response)
	if err != nil {
		log.Printf("Error encode %s", err.Error())
		return
	}

	// 4. response
	syscall.Write(fd, []byte(encodedRes))
}

func handleCommand(decodeRequest any) (any, error) {
	arr, ok := decodeRequest.([]any)
	if !ok || decodeRequest == nil || len(arr) == 0 {
		return "", errors.New("invalid command")
	}

	cmdName, cmd := convertToCommand(arr)

	switch cmdName {
	case "PING":
		return core.HandlePing(cmd)
	case "GET":
		return core.HandleGet(cmd)
	case "SET":
		return core.HandleSet(cmd)
	case "TTL":
		return core.HandleTTL(cmd)
	case "EXPIRE":
		return core.HandleExpire(cmd)
	case "DEL":
		return core.HandleDel(cmd)
	case "EXISTS":
		return core.HandleExists(cmd)
	case "SADD":
		return core.HandleSetAdd(cmd)
	case "SISMEMBER":
		return core.HandleSISMEMBER(cmd)
	case "SREM":
		return core.HandleSREM(cmd)
	case "SMEMBERS":
		return core.HandleSMEMBERS(cmd)
	case "FLUSHDB":
		return core.HandleFlushDb(cmd)
	case "PERSIST":
		return core.HandlePERSIST(cmd)
	default:
		return nil, errors.New("invalid command")

	}
}

func convertToCommand(arr []any) (string, *core.Command) {
	cmdName := arr[0].(string)
	cmdName = strings.ToUpper(cmdName)
	var args []string
	for i := 1; i < len(arr); i++ {
		args = append(args, fmt.Sprintf("%v", arr[i]))
	}

	cmd := core.NewCommand(cmdName, args)
	return cmdName, cmd
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
