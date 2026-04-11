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
	"github.com/manh119/Redis/internal/core/io_multiplexing"
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

	// 3. wait for new event in the epoll, event loop
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
				handleNewConnection(multiplexer, fdListener)
			} else {
				handleClientCommand(multiplexer, currentFd)
			}
		}
	}
}

func handleNewConnection(multiplexer *io_multiplexing.Epoll, fdListener int) {
	fdConn, connAdrr, err := syscall.Accept(fdListener)
	ip, port := parseSockaddr(connAdrr)
	log.Printf("new connection fd=%d from %s:%d", fdConn, ip, port)
	if err != nil {
		log.Printf("error connect to %s with error : %s", connAdrr, err.Error())
		return
	}
	syscall.SetNonblock(fdConn, true)

	// 4. add new connection to the epoll to monitor
	err = multiplexer.Monitor(io_multiplexing.Event{
		fdConn, io_multiplexing.OpRead,
	})
	if err != nil {
		log.Printf(err.Error())
	}
}

func handleClientCommand(multiplexer *io_multiplexing.Epoll, fd int) {
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

func handleCleanup(multiplexer *io_multiplexing.Epoll, ln net.Listener) {
	log.Println("Saving data to the disk (Persistence)...")
	//core.SaveRDB()

	log.Println("Closing connection...")
	multiplexer.Close()
	ln.Close()
}

func HandleShutDown(sig <-chan os.Signal, wg *sync.WaitGroup) {
	defer wg.Done()
	<-sig
	atomic.StoreInt32(&ServerStatus, config.ServerStatusShuttingDown)
}
