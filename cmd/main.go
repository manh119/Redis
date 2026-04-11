package main

import (
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/manh119/Redis/server"
)

func main() {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)

	wg := sync.WaitGroup{}
	wg.Add(100)

	go server.RunIoMultiplexingServerMultipleIOHanlder(&wg)
	go server.HandleShutDown(sig, &wg)

	wg.Wait()
	println("Server shutdown successfully. Bye bye.")
}
