package main

import (
	"log"
	"os"
	"os/signal"

	"github.com/manh119/Redis/internal/core/server"
)

func main() {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)

	go server.RunIoMultiplexingServer()

	<-sig
	log.Println("interrupted received")
}
