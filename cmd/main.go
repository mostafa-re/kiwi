package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"kv-service/internal/api"
	"kv-service/internal/config"
	"kv-service/internal/replication"
	"kv-service/internal/storage"
)

func main() {
	// Load configuration
	cfg := config.Load()

	log.Printf("Starting KV Service %s (commit: %s)", cfg.Version, cfg.GitCommit)
	log.Printf("Node ID: %s, Role: %s", cfg.NodeID, cfg.Role)

	// Initialize base storage layer
	baseStore, err := storage.NewLevelDBStore(cfg.DatabasePath)
	if err != nil {
		log.Fatalf("Failed to initialize storage: %v", err)
	}

	// Initialize replication components
	var replManager *replication.Manager
	var replServer *replication.Server

	if cfg.IsMaster() {
		// Master: connect to slaves for replication
		log.Printf("Initializing as MASTER with slaves: %v", cfg.SlaveAddrs)

		// Wait a bit for slaves to start (in production, use retry logic)
		if len(cfg.SlaveAddrs) > 0 {
			log.Printf("Waiting for slaves to be ready...")
			time.Sleep(2 * time.Second)
		}

		replManager = replication.NewManager(cfg.SlaveAddrs)
	}

	// All nodes run gRPC server (for health checks, and slaves for replication)
	replServer = replication.NewServer(cfg, baseStore)
	if err := replServer.Start(); err != nil {
		log.Fatalf("Failed to start replication server: %v", err)
	}

	// Create replicated store wrapper
	store := storage.NewReplicatedStore(baseStore, cfg, replManager)

	// Initialize and configure HTTP server
	server := api.NewServer(cfg, store)

	// Setup graceful shutdown
	go handleShutdown(server, replServer, store)

	// Start HTTP server
	log.Printf("HTTP server starting on port %s", cfg.Port)
	if err := server.Start(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func handleShutdown(server *api.Server, replServer *replication.Server, store *storage.ReplicatedStore) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down server...")

	if replServer != nil {
		replServer.Stop()
	}

	if err := server.Shutdown(); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}

	if err := store.Close(); err != nil {
		log.Printf("Storage close error: %v", err)
	}
}
