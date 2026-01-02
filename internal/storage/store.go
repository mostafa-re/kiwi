package storage

import "errors"

var (
	// ErrKeyNotFound is returned when a key does not exist
	ErrKeyNotFound = errors.New("key not found")

	// ErrInvalidKey is returned when a key is invalid
	ErrInvalidKey = errors.New("invalid key")
)

// Store defines the interface for key-value storage operations
type Store interface {
	// Put stores a key-value pair in the specified collection
	Put(collection, key string, value interface{}) error

	// Get retrieves a value by key from the specified collection
	Get(collection, key string) (interface{}, error)

	// Delete removes a key from the specified collection
	Delete(collection, key string) error

	// List returns all key-value pairs in the specified collection
	List(collection string) (map[string]interface{}, error)

	// Count returns the number of keys in a collection
	Count(collection string) (int, error)

	// ListCollections returns all available collections
	ListCollections() ([]string, error)

	// Close closes the store connection
	Close() error
}

// BatchOperations defines the interface for batch operations
type BatchOperations interface {
	// Put adds a put operation to the batch
	Put(collection, key string, value interface{}) error

	// Delete adds a delete operation to the batch
	Delete(collection, key string)

	// Commit executes all operations in the batch atomically
	Commit() error
}
