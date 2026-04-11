package main

import (
	"log"
	"net/http"
	_ "net/http/pprof"
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
	wg.Add(1)

	go server.RunIoMultiplexingServer(&wg)
	go server.HandleShutDown(sig, &wg)

	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()

	wg.Wait()
	println("Server shutdown successfully. Bye bye.")
}
