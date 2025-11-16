package main

import "redis-clone/internal/server"

func main() {
	server.RunIoMultiplexingServer()
}
