package replication

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	pb "kiwi/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Client handles outgoing replication requests to a single slave
type Client struct {
	addr   string
	conn   *grpc.ClientConn
	client pb.ReplicationServiceClient
	mu     sync.RWMutex
}

// NewClient creates a new replication client for a slave
func NewClient(addr string) (*Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to slave %s: %w", addr, err)
	}

	return &Client{
		addr:   addr,
		conn:   conn,
		client: pb.NewReplicationServiceClient(conn),
	}, nil
}

// Close closes the connection
func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// Prepare sends Phase 1 of 2PC to the slave
func (c *Client) Prepare(ctx context.Context, txnID string, op pb.OperationType, collection, key string, value []byte, seq uint64) (bool, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	resp, err := c.client.Prepare(ctx, &pb.PrepareRequest{
		TransactionId: txnID,
		Operation:     op,
		Collection:    collection,
		Key:           key,
		Value:         value,
		Sequence:      seq,
	})
	if err != nil {
		return false, fmt.Errorf("prepare to %s failed: %w", c.addr, err)
	}

	return resp.Ready, nil
}

// Commit sends Phase 2 commit to the slave
func (c *Client) Commit(ctx context.Context, txnID string) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	resp, err := c.client.Commit(ctx, &pb.CommitRequest{
		TransactionId: txnID,
	})
	if err != nil {
		return fmt.Errorf("commit to %s failed: %w", c.addr, err)
	}

	if !resp.Success {
		return fmt.Errorf("commit to %s rejected: %s", c.addr, resp.Error)
	}

	return nil
}

// Abort sends Phase 2 abort to the slave
func (c *Client) Abort(ctx context.Context, txnID string) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	_, err := c.client.Abort(ctx, &pb.AbortRequest{
		TransactionId: txnID,
	})
	if err != nil {
		return fmt.Errorf("abort to %s failed: %w", c.addr, err)
	}

	return nil
}

// HealthCheck checks if the slave is healthy
func (c *Client) HealthCheck(ctx context.Context) (*pb.HealthCheckResponse, error) {
	return c.client.HealthCheck(ctx, &pb.HealthCheckRequest{})
}

// Address returns the slave address
func (c *Client) Address() string {
	return c.addr
}

// Manager manages replication to all slaves using 2PC (runs on master)
type Manager struct {
	clients []*Client
	seq     uint64
	txnID   uint64
	mu      sync.Mutex
}

// NewManager creates a new replication manager
func NewManager(slaveAddrs []string) *Manager {
	m := &Manager{
		clients: make([]*Client, 0),
	}

	for _, addr := range slaveAddrs {
		if addr == "" {
			continue
		}
		client, err := NewClient(addr)
		if err != nil {
			log.Printf("[Replication] Warning: failed to connect to slave %s: %v", addr, err)
			continue
		}
		m.clients = append(m.clients, client)
		log.Printf("[Replication] Connected to slave: %s", addr)
	}

	return m
}

// Close closes all client connections
func (m *Manager) Close() {
	for _, client := range m.clients {
		client.Close()
	}
}

// generateTxnID generates a unique transaction ID
func (m *Manager) generateTxnID() string {
	id := atomic.AddUint64(&m.txnID, 1)
	return fmt.Sprintf("txn-%d-%d", time.Now().UnixNano(), id)
}

// nextSeq generates the next sequence number
func (m *Manager) nextSeq() uint64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.seq++
	return m.seq
}

// ReplicatePut replicates a PUT operation using 2PC
func (m *Manager) ReplicatePut(collection, key string, value []byte) error {
	return m.replicate2PC(pb.OperationType_PUT, collection, key, value)
}

// ReplicateDelete replicates a DELETE operation using 2PC
func (m *Manager) ReplicateDelete(collection, key string) error {
	return m.replicate2PC(pb.OperationType_DELETE, collection, key, nil)
}

// replicate2PC performs Two-Phase Commit across all slaves
func (m *Manager) replicate2PC(op pb.OperationType, collection, key string, value []byte) error {
	if len(m.clients) == 0 {
		return nil
	}

	txnID := m.generateTxnID()
	seq := m.nextSeq()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	log.Printf("[2PC] Starting transaction %s: op=%v collection=%s key=%s", txnID, op, collection, key)

	// ==================== PHASE 1: PREPARE ====================
	// Send prepare to all slaves in parallel
	type prepareResult struct {
		client *Client
		ready  bool
		err    error
	}

	prepareChan := make(chan prepareResult, len(m.clients))
	for _, client := range m.clients {
		go func(c *Client) {
			ready, err := c.Prepare(ctx, txnID, op, collection, key, value, seq)
			prepareChan <- prepareResult{client: c, ready: ready, err: err}
		}(client)
	}

	// Collect prepare responses
	allReady := true
	preparedClients := make([]*Client, 0, len(m.clients))
	var prepareErrors []error

	for i := 0; i < len(m.clients); i++ {
		result := <-prepareChan
		if result.err != nil {
			allReady = false
			prepareErrors = append(prepareErrors, result.err)
			log.Printf("[2PC] PREPARE failed for %s: %v", result.client.Address(), result.err)
		} else if !result.ready {
			allReady = false
			log.Printf("[2PC] PREPARE rejected by %s", result.client.Address())
		} else {
			preparedClients = append(preparedClients, result.client)
			log.Printf("[2PC] PREPARE successful for %s", result.client.Address())
		}
	}

	// ==================== PHASE 2: COMMIT or ABORT ====================
	if !allReady {
		// Abort all prepared slaves
		log.Printf("[2PC] Transaction %s: aborting due to prepare failure", txnID)
		m.abortAll(ctx, txnID, preparedClients)

		if len(prepareErrors) > 0 {
			return fmt.Errorf("2PC prepare failed: %v", prepareErrors[0])
		}
		return fmt.Errorf("2PC prepare rejected by one or more slaves")
	}

	// All slaves are ready - send commit to all
	log.Printf("[2PC] Transaction %s: all slaves ready, committing", txnID)

	commitChan := make(chan error, len(m.clients))
	for _, client := range m.clients {
		go func(c *Client) {
			commitChan <- c.Commit(ctx, txnID)
		}(client)
	}

	// Collect commit responses
	var commitErrors []error
	for i := 0; i < len(m.clients); i++ {
		if err := <-commitChan; err != nil {
			commitErrors = append(commitErrors, err)
			log.Printf("[2PC] COMMIT failed: %v", err)
		}
	}

	if len(commitErrors) > 0 {
		// This is a serious issue - some slaves committed, some didn't
		// In production, we'd need recovery mechanisms
		log.Printf("[2PC] WARNING: Transaction %s partially committed! Errors: %v", txnID, commitErrors)
		return fmt.Errorf("2PC commit partially failed: %v", commitErrors[0])
	}

	log.Printf("[2PC] Transaction %s: committed successfully to %d slaves", txnID, len(m.clients))
	return nil
}

// abortAll sends abort to all prepared slaves
func (m *Manager) abortAll(ctx context.Context, txnID string, clients []*Client) {
	var wg sync.WaitGroup
	for _, client := range clients {
		wg.Add(1)
		go func(c *Client) {
			defer wg.Done()
			if err := c.Abort(ctx, txnID); err != nil {
				log.Printf("[2PC] ABORT failed for %s: %v", c.Address(), err)
			} else {
				log.Printf("[2PC] ABORT successful for %s", c.Address())
			}
		}(client)
	}
	wg.Wait()
}

// SlaveCount returns the number of connected slaves
func (m *Manager) SlaveCount() int {
	return len(m.clients)
}

// HealthCheckAll checks health of all slaves
func (m *Manager) HealthCheckAll() map[string]bool {
	results := make(map[string]bool)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	for _, client := range m.clients {
		_, err := client.HealthCheck(ctx)
		results[client.Address()] = err == nil
	}

	return results
}
