package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"kv-service/internal/api"
	"kv-service/internal/config"
	"kv-service/internal/storage"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Initialize storage layer
	store, err := storage.NewLevelDBStore(cfg.DatabasePath)
	if err != nil {
		log.Fatalf("Failed to initialize storage: %v", err)
	}
	defer store.Close()

	// Initialize and configure HTTP server
	server := api.NewServer(cfg, store)

	// Setup graceful shutdown
	go handleShutdown(server)

	// Start HTTP server
	log.Printf("Server starting on port %s", cfg.Port)
	if err := server.Start(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func handleShutdown(server *api.Server) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down server...")
	if err := server.Shutdown(); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}
}
