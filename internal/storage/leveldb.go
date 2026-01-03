package storage

import (
	"encoding/json"
	"fmt"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
)

// LevelDBStore implements the Store interface using LevelDB
type LevelDBStore struct {
	db *leveldb.DB
}

// NewLevelDBStore creates a new LevelDB-backed store
func NewLevelDBStore(path string) (*LevelDBStore, error) {
	db, err := leveldb.OpenFile(path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	return &LevelDBStore{db: db}, nil
}

// Close closes the database connection
func (s *LevelDBStore) Close() error {
	return s.db.Close()
}

// PutDirect stores raw bytes directly (used for replication)
func (s *LevelDBStore) PutDirect(collection, key string, value []byte) error {
	if key == "" {
		return ErrInvalidKey
	}
	dbKey := s.makeKey(collection, key)
	return s.db.Put([]byte(dbKey), value, nil)
}

// DeleteDirect deletes a key directly (used for replication)
func (s *LevelDBStore) DeleteDirect(collection, key string) error {
	if key == "" {
		return ErrInvalidKey
	}
	dbKey := s.makeKey(collection, key)
	return s.db.Delete([]byte(dbKey), nil)
}

// makeKey creates a namespaced key with collection prefix
func (s *LevelDBStore) makeKey(collection, key string) string {
	return fmt.Sprintf("%s:%s", collection, key)
}

// Put stores a key-value pair in the specified collection
func (s *LevelDBStore) Put(collection, key string, value interface{}) error {
	if key == "" {
		return ErrInvalidKey
	}

	// Serialize value to JSON
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to serialize value: %w", err)
	}

	// Store in LevelDB with collection prefix
	dbKey := s.makeKey(collection, key)
	if err := s.db.Put([]byte(dbKey), data, nil); err != nil {
		return fmt.Errorf("failed to store value: %w", err)
	}

	return nil
}

// Get retrieves a value by key from the specified collection
func (s *LevelDBStore) Get(collection, key string) (interface{}, error) {
	if key == "" {
		return nil, ErrInvalidKey
	}

	dbKey := s.makeKey(collection, key)
	data, err := s.db.Get([]byte(dbKey), nil)
	if err != nil {
		if err == leveldb.ErrNotFound {
			return nil, ErrKeyNotFound
		}
		return nil, fmt.Errorf("failed to retrieve value: %w", err)
	}

	// Deserialize JSON
	var value interface{}
	if err := json.Unmarshal(data, &value); err != nil {
		return nil, fmt.Errorf("failed to deserialize value: %w", err)
	}

	return value, nil
}

// Delete removes a key from the specified collection
func (s *LevelDBStore) Delete(collection, key string) error {
	if key == "" {
		return ErrInvalidKey
	}

	dbKey := s.makeKey(collection, key)

	// Check if key exists
	_, err := s.db.Get([]byte(dbKey), nil)
	if err != nil {
		if err == leveldb.ErrNotFound {
			return ErrKeyNotFound
		}
		return fmt.Errorf("failed to check key existence: %w", err)
	}

	// Delete the key
	if err := s.db.Delete([]byte(dbKey), nil); err != nil {
		return fmt.Errorf("failed to delete key: %w", err)
	}

	return nil
}

// List returns all key-value pairs in the specified collection
func (s *LevelDBStore) List(collection string) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	// Create prefix for the collection
	prefix := []byte(collection + ":")

	// Create iterator for the collection prefix
	iter := s.db.NewIterator(util.BytesPrefix(prefix), nil)
	defer iter.Release()

	// Iterate through all keys with the prefix
	for iter.Next() {
		key := string(iter.Key())
		value := iter.Value()

		// Remove collection prefix from key
		actualKey := key[len(collection)+1:]

		// Deserialize value
		var val interface{}
		if err := json.Unmarshal(value, &val); err != nil {
			// Skip malformed entries
			continue
		}

		result[actualKey] = val
	}

	if err := iter.Error(); err != nil {
		return nil, fmt.Errorf("iterator error: %w", err)
	}

	return result, nil
}

// ListCollections returns all available collections
func (s *LevelDBStore) ListCollections() ([]string, error) {
	collections := make(map[string]bool)

	iter := s.db.NewIterator(nil, nil)
	defer iter.Release()

	for iter.Next() {
		key := string(iter.Key())
		// Extract collection name (everything before first colon)
		for i, c := range key {
			if c == ':' {
				collections[key[:i]] = true
				break
			}
		}
	}

	if err := iter.Error(); err != nil {
		return nil, fmt.Errorf("iterator error: %w", err)
	}

	result := make([]string, 0, len(collections))
	for collection := range collections {
		result = append(result, collection)
	}

	return result, nil
}

// Count returns the number of keys in a collection
func (s *LevelDBStore) Count(collection string) (int, error) {
	count := 0
	prefix := []byte(collection + ":")

	iter := s.db.NewIterator(util.BytesPrefix(prefix), nil)
	defer iter.Release()

	for iter.Next() {
		count++
	}

	if err := iter.Error(); err != nil {
		return 0, fmt.Errorf("iterator error: %w", err)
	}

	return count, nil
}

// LevelDBBatch represents a batch of write operations
type LevelDBBatch struct {
	store *LevelDBStore
	batch *leveldb.Batch
}

// NewBatch creates a new batch for atomic writes
func (s *LevelDBStore) NewBatch() *LevelDBBatch {
	return &LevelDBBatch{
		store: s,
		batch: new(leveldb.Batch),
	}
}

// Put adds a put operation to the batch
func (b *LevelDBBatch) Put(collection, key string, value interface{}) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to serialize value: %w", err)
	}

	dbKey := b.store.makeKey(collection, key)
	b.batch.Put([]byte(dbKey), data)
	return nil
}

// Delete adds a delete operation to the batch
func (b *LevelDBBatch) Delete(collection, key string) {
	dbKey := b.store.makeKey(collection, key)
	b.batch.Delete([]byte(dbKey))
}

// Commit executes all operations in the batch atomically
func (b *LevelDBBatch) Commit() error {
	return b.store.db.Write(b.batch, nil)
}
