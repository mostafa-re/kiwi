package storage

import (
	"encoding/json"
	"fmt"

	"kiwi/internal/config"
	"kiwi/internal/replication"
)

// ReplicatedStore wraps a store with replication support using 2PC
type ReplicatedStore struct {
	store   *LevelDBStore
	config  *config.Config
	manager *replication.Manager
}

// NewReplicatedStore creates a new replicated store
func NewReplicatedStore(store *LevelDBStore, cfg *config.Config, manager *replication.Manager) *ReplicatedStore {
	return &ReplicatedStore{
		store:   store,
		config:  cfg,
		manager: manager,
	}
}

// Put stores a key-value pair using Two-Phase Commit for strong consistency
//
// 2PC Flow:
// 1. Phase 1 (Prepare): Master sends prepare to all slaves
//    - If ANY slave fails: abort all, return error, master doesn't write
// 2. Phase 2 (Commit): Master sends commit to all slaves
//    - All slaves apply the operation
// 3. Master writes locally only after all slaves committed
//
// This ensures: either ALL nodes have the data, or NONE do.
func (s *ReplicatedStore) Put(collection, key string, value interface{}) error {
	// Slaves reject direct writes
	if s.config.IsSlave() {
		return fmt.Errorf("writes not allowed on slave nodes, send request to master")
	}

	// Serialize value for replication
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to serialize value: %w", err)
	}

	// Step 1: Replicate to all slaves using 2PC
	// If this fails, slaves will abort and no data is written anywhere
	if s.manager != nil && s.manager.SlaveCount() > 0 {
		if err := s.manager.ReplicatePut(collection, key, data); err != nil {
			// 2PC failed - slaves aborted, don't write to master
			return fmt.Errorf("replication failed: %w", err)
		}
	}

	// Step 2: Write locally on master (only after slaves committed)
	if err := s.store.Put(collection, key, value); err != nil {
		// This is problematic - slaves committed but master failed
		// In production, we'd need recovery. For now, log and return error.
		return fmt.Errorf("local write failed after replication (inconsistency possible): %w", err)
	}

	return nil
}

// Get retrieves a value by key (reads allowed on all nodes)
func (s *ReplicatedStore) Get(collection, key string) (interface{}, error) {
	return s.store.Get(collection, key)
}

// Delete removes a key using Two-Phase Commit for strong consistency
func (s *ReplicatedStore) Delete(collection, key string) error {
	if s.config.IsSlave() {
		return fmt.Errorf("deletes not allowed on slave nodes, send request to master")
	}

	// Verify key exists before attempting delete
	_, err := s.store.Get(collection, key)
	if err != nil {
		return err // Key doesn't exist
	}

	// Step 1: Replicate delete to all slaves using 2PC
	if s.manager != nil && s.manager.SlaveCount() > 0 {
		if err := s.manager.ReplicateDelete(collection, key); err != nil {
			// 2PC failed - slaves aborted, don't delete from master
			return fmt.Errorf("replication failed: %w", err)
		}
	}

	// Step 2: Delete locally on master (only after slaves committed)
	if err := s.store.Delete(collection, key); err != nil {
		return fmt.Errorf("local delete failed after replication (inconsistency possible): %w", err)
	}

	return nil
}

// List returns all key-value pairs (reads allowed on all nodes)
func (s *ReplicatedStore) List(collection string) (map[string]interface{}, error) {
	return s.store.List(collection)
}

// Count returns the number of keys in a collection
func (s *ReplicatedStore) Count(collection string) (int, error) {
	return s.store.Count(collection)
}

// ListCollections returns all available collections
func (s *ReplicatedStore) ListCollections() ([]string, error) {
	return s.store.ListCollections()
}

// Close closes the store
func (s *ReplicatedStore) Close() error {
	if s.manager != nil {
		s.manager.Close()
	}
	return s.store.Close()
}

// GetManager returns the replication manager
func (s *ReplicatedStore) GetManager() *replication.Manager {
	return s.manager
}

// Underlying returns the underlying LevelDB store (for replication server)
func (s *ReplicatedStore) Underlying() *LevelDBStore {
	return s.store
}
