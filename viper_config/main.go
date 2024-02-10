package main

import (
	"log"
	"os"
)

func main() {
	os.Setenv("SERVER_PORT", "8081")
	config := Setup()
	log.Printf("Server: %s:%d", config.Server.Host, config.Server.Port)
	log.Printf("Timeout: %s", config.Server.Timeout)
}
