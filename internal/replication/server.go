package replication

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"

	"kiwi/internal/config"
	pb "kiwi/proto"

	"google.golang.org/grpc"
)

// StorageBackend interface for the replication server to write data
type StorageBackend interface {
	PutDirect(collection, key string, value []byte) error
	DeleteDirect(collection, key string) error
}

// PendingTransaction holds a prepared but not yet committed transaction
type PendingTransaction struct {
	Operation  pb.OperationType
	Collection string
	Key        string
	Value      []byte
}

// Server handles incoming replication requests (runs on slaves)
// Implements Two-Phase Commit (2PC) for strong consistency
type Server struct {
	pb.UnimplementedReplicationServiceServer
	config   *config.Config
	storage  StorageBackend
	server   *grpc.Server
	mu       sync.RWMutex
	seq      uint64
	pending  map[string]*PendingTransaction // transaction_id -> pending transaction
}

// NewServer creates a new replication gRPC server
func NewServer(cfg *config.Config, storage StorageBackend) *Server {
	return &Server{
		config:  cfg,
		storage: storage,
		pending: make(map[string]*PendingTransaction),
	}
}

// Start starts the gRPC server
func (s *Server) Start() error {
	lis, err := net.Listen("tcp", ":"+s.config.GRPCPort)
	if err != nil {
		return fmt.Errorf("failed to listen on port %s: %w", s.config.GRPCPort, err)
	}

	s.server = grpc.NewServer()
	pb.RegisterReplicationServiceServer(s.server, s)

	log.Printf("[Replication] gRPC server starting on port %s (role: %s)", s.config.GRPCPort, s.config.Role)

	go func() {
		if err := s.server.Serve(lis); err != nil {
			log.Printf("[Replication] gRPC server error: %v", err)
		}
	}()

	return nil
}

// Stop gracefully stops the gRPC server
func (s *Server) Stop() {
	if s.server != nil {
		s.server.GracefulStop()
	}
}

// Prepare handles Phase 1 of 2PC - validate and stage the operation
func (s *Server) Prepare(ctx context.Context, req *pb.PrepareRequest) (*pb.PrepareResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	log.Printf("[2PC] PREPARE received: txn=%s op=%v collection=%s key=%s",
		req.TransactionId, req.Operation, req.Collection, req.Key)

	// Check if transaction already exists (duplicate prepare)
	if _, exists := s.pending[req.TransactionId]; exists {
		log.Printf("[2PC] Transaction %s already prepared", req.TransactionId)
		return &pb.PrepareResponse{Ready: true}, nil
	}

	// Validate the operation (check if we can perform it)
	// For PUT: we can always prepare
	// For DELETE: check if key exists (optional, we can skip this for simplicity)

	// Stage the transaction
	s.pending[req.TransactionId] = &PendingTransaction{
		Operation:  req.Operation,
		Collection: req.Collection,
		Key:        req.Key,
		Value:      req.Value,
	}

	log.Printf("[2PC] PREPARE successful: txn=%s - ready to commit", req.TransactionId)
	return &pb.PrepareResponse{Ready: true}, nil
}

// Commit handles Phase 2 of 2PC - apply the staged operation
func (s *Server) Commit(ctx context.Context, req *pb.CommitRequest) (*pb.CommitResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	log.Printf("[2PC] COMMIT received: txn=%s", req.TransactionId)

	// Get the pending transaction
	txn, exists := s.pending[req.TransactionId]
	if !exists {
		log.Printf("[2PC] COMMIT failed: transaction %s not found", req.TransactionId)
		return &pb.CommitResponse{Success: false, Error: "transaction not found"}, nil
	}

	// Apply the operation
	var err error
	switch txn.Operation {
	case pb.OperationType_PUT:
		err = s.storage.PutDirect(txn.Collection, txn.Key, txn.Value)
	case pb.OperationType_DELETE:
		err = s.storage.DeleteDirect(txn.Collection, txn.Key)
	}

	if err != nil {
		log.Printf("[2PC] COMMIT failed: txn=%s error=%v", req.TransactionId, err)
		// Don't remove from pending - might retry
		return &pb.CommitResponse{Success: false, Error: err.Error()}, nil
	}

	// Remove from pending
	delete(s.pending, req.TransactionId)

	log.Printf("[2PC] COMMIT successful: txn=%s", req.TransactionId)
	return &pb.CommitResponse{Success: true}, nil
}

// Abort handles Phase 2 of 2PC - discard the staged operation
func (s *Server) Abort(ctx context.Context, req *pb.AbortRequest) (*pb.AbortResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	log.Printf("[2PC] ABORT received: txn=%s", req.TransactionId)

	// Remove from pending (discard the staged operation)
	delete(s.pending, req.TransactionId)

	log.Printf("[2PC] ABORT successful: txn=%s", req.TransactionId)
	return &pb.AbortResponse{Success: true}, nil
}

// HealthCheck responds to health check requests
func (s *Server) HealthCheck(ctx context.Context, req *pb.HealthCheckRequest) (*pb.HealthCheckResponse, error) {
	return &pb.HealthCheckResponse{
		Healthy: true,
		NodeId:  s.config.NodeID,
		Role:    string(s.config.Role),
	}, nil
}
