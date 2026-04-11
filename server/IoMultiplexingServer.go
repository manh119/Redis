package server

import (
	"log"
	"net"
	"os"
	"sync"
	"sync/atomic"
	"syscall"

	"github.com/manh119/Redis/internal/config"
	"github.com/manh119/Redis/internal/core/command"
	"github.com/manh119/Redis/internal/core/resp"
)

var ServerStatus int32 = config.ServerStatusIdle

func RunIoMultiplexingServer(wg *sync.WaitGroup) {
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
	log.Printf("Server is listening on port%s ...", config.Port)

	// 3. wait for new event in the epoll, event loop
	bufferEvents := make([]syscall.EpollEvent, 100)
	for {
		if atomic.LoadInt32(&ServerStatus) == config.ServerStatusShuttingDown {
			handleCleanup(fdEpoll, listener)
			return
		}

		n, err := syscall.EpollWait(fdEpoll, bufferEvents, 100)
		if err != nil {
			log.Printf(err.Error())
			continue
		}

		for i := 0; i < n; i++ {
			currentFd := bufferEvents[i].Fd
			if currentFd == int32(fdListener) {
				handleNewConnection(fdListener, fdEpoll)
			} else {
				handleClientCommand(fdEpoll, int(currentFd))
			}
		}
	}
}

func handleNewConnection(fdListener int, fdEpoll int) {
	fdConn, connAdrr, err := syscall.Accept(fdListener)
	ip, port := parseSockaddr(connAdrr)
	log.Printf("new connection fd=%d from %s:%d", fdConn, ip, port)
	if err != nil {
		log.Printf("error connect to %s with error : %s", connAdrr, err.Error())
		return
	}
	syscall.SetNonblock(fdConn, true)

	// 4. add new connection to the epoll to monitor
	connEvent := &syscall.EpollEvent{Events: syscall.EPOLLIN, Fd: int32(fdConn)}
	err = syscall.EpollCtl(fdEpoll, syscall.EPOLL_CTL_ADD, fdConn, connEvent)
	if err != nil {
		log.Printf(err.Error())
	}
}

func handleClientCommand(fdEpoll int, fd int) {
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
	log.Printf("decodedMess: %s", decodeRequest)

	// 2. read command
	response, err := command.HandleCommand(decodeRequest)
	if err != nil {
		log.Printf(err.Error())
		if err.Error() == config.NILL {
			syscall.Write(fd, []byte(config.NILL))
			return
		}
		resError, err := resp.Encode(err)
		if err != nil {
			log.Printf("Error encode %s", err.Error())
			return
		}
		syscall.Write(fd, []byte(resError))
		return
	}

	// 3. encode response
	encodedRes, err := resp.Encode(response)
	if err != nil {
		log.Printf("Error encode %s", err.Error())
		return
	}

	// 4. response
	log.Printf("encodeMess: %s", encodedRes)
	syscall.Write(fd, []byte(encodedRes))
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

func handleCleanup(fdEp int, ln net.Listener) {
	log.Println("Saving data to the disk (Persistence)...")
	//core.SaveRDB()

	log.Println("Closing connection...")
	//mux.Close()
	syscall.Close(fdEp)
	ln.Close()
}

func HandleShutDown(sig <-chan os.Signal, wg *sync.WaitGroup) {
	defer wg.Done()
	<-sig
	atomic.StoreInt32(&ServerStatus, config.ServerStatusShuttingDown)
}
