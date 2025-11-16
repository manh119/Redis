package main

import "redis-clone/server"

func main() {
	server.RunIoMultiplexingServer()
}
