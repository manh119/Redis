package main

import (
	"log"
	"os"
	"os/signal"
	"sync"

	"github.com/manh119/Redis/server"
)

func main() {
	wg := sync.WaitGroup{}
	wg.Add(1)
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)

	go server.RunIoMultiplexingServer(&wg)

	<-sig
	log.Println("shutdown signal received")
	server.ShutdownAsap.Store(1)
	wg.Wait()
}
