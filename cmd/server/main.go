package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/jackdar/go-chat/internal/server"
)

func main() {
	config := server.ParseConfig()

	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("Starting chat server...")

	server := server.NewServer(config)

	if err := server.Start(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}

	log.Printf("Server listening on %s", config.Address())
	log.Println("Press Ctrl+C to stop")

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	<-shutdown
	log.Println("\nShutting down gracefully...")

	if err := server.Stop(); err != nil {
		log.Printf("Error during shutdown: %v", err)
	}

	log.Println("Server stopped")
}
