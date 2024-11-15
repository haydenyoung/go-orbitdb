package storage

// Storage is an interface
type Storage interface {
	// Put stores a key-value pair in the storage.
	Put(key string, value []byte) error

	// Get retrieves a value by its key. Returns an error if the key is not found.
	Get(key string) ([]byte, error)

	// Delete removes a key-value pair from the storage.
	Delete(key string) error

	// Iterator returns a channel that yields key-value pairs.
	Iterator() (<-chan [2]string, error)

	// Merge merges data from another storage instance.
	Merge(other Storage) error

	// Clear removes all key-value pairs from the storage.
	Clear() error

	// Close closes the storage and releases resources.
	Close() error
}
