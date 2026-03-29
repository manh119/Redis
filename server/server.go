package server

import (
	"log"
	"net"
	"syscall"
)

func RunIoMultiplexingServer() {
	listener, _ := net.Listen("tcp", ":4000")
	defer listener.Close()

	file, _ := listener.(*net.TCPListener).File()
	fdListener := int(file.Fd())
	syscall.SetNonblock(fdListener, true)

	// 1. create epoll instance
	fdEpoll, _ := syscall.EpollCreate1(0)
	defer syscall.Close(fdEpoll)

	// 2. add tcp connection to the epoll to handle new connection
	listenerEvent := &syscall.EpollEvent{Events: syscall.EPOLLIN, Fd: int32(fdListener)}
	_ = syscall.EpollCtl(fdEpoll, syscall.EPOLL_CTL_ADD, fdListener, listenerEvent)

	// 3. wait for new event in the epoll, event loop
	bufferEvents := make([]syscall.EpollEvent, 100)
	for {
		n, err := syscall.EpollWait(fdEpoll, bufferEvents, 100)
		if err != nil {
			log.Printf(err.Error())
			continue
		}

		for i := 0; i < n; i++ {
			currentFd := bufferEvents[i].Fd
			if currentFd == int32(fdListener) {
				fdConn, connAdrr, err := syscall.Accept(fdListener)
				log.Printf("new connection from fd : %d with new addrr %s", currentFd, connAdrr)
				if err != nil {
					return
				}

				// 4. add new connection to the epoll to monitor
				connEvent := &syscall.EpollEvent{Events: syscall.EPOLLIN, Fd: int32(fdConn)}
				err = syscall.EpollCtl(fdEpoll, syscall.EPOLL_CTL_ADD, fdConn, connEvent)
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
	if err != nil {
		syscall.EpollCtl(fdEpoll, syscall.EPOLL_CTL_DEL, fd, nil)
		syscall.Close(fd)
		return
	}

	log.Printf("read %d bytes from %d", n, fd)
	syscall.Write(fd, []byte("PONG\n"))
}
