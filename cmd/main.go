package main

import (
	"github.com/manh119/Redis/internal/core/server"
)

func main() {
	server.RunIoMultiplexingServer()
}
