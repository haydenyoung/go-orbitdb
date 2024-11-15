package storage

import (
	"errors"
	lru "github.com/hashicorp/golang-lru"
)

// LRUStorage implements the Storage interface using an LRU cache.
type LRUStorage struct {
	cache *lru.Cache
}

// NewLRUStorage initializes an LRUStorage with a given size.
func NewLRUStorage(size int) (*LRUStorage, error) {
	cache, err := lru.New(size)
	if err != nil {
		return nil, err
	}
	return &LRUStorage{cache: cache}, nil
}

// Put stores a key-value pair in the LRU cache.
func (s *LRUStorage) Put(key string, value []byte) error {
	s.cache.Add(key, value)
	return nil
}

// Get retrieves a value by its key from the LRU cache.
func (s *LRUStorage) Get(key string) ([]byte, error) {
	if value, ok := s.cache.Get(key); ok {
		return value.([]byte), nil
	}
	return nil, errors.New("key not found")
}

// Delete removes a key-value pair from the LRU cache.
func (s *LRUStorage) Delete(key string) error {
	s.cache.Remove(key)
	return nil
}

// Iterator returns a channel that yields key-value pairs.
func (s *LRUStorage) Iterator() (<-chan [2]string, error) {
	ch := make(chan [2]string)
	go func() {
		defer close(ch)
		for _, key := range s.cache.Keys() {
			value, _ := s.cache.Get(key)
			ch <- [2]string{key.(string), string(value.([]byte))}
		}
	}()
	return ch, nil
}

// Merge merges data from another storage instance.
func (s *LRUStorage) Merge(other Storage) error {
	iter, err := other.Iterator()
	if err != nil {
		return err
	}
	for kv := range iter {
		s.cache.Add(kv[0], []byte(kv[1]))
	}
	return nil
}

// Clear removes all key-value pairs from the LRU cache.
func (s *LRUStorage) Clear() error {
	s.cache.Purge()
	return nil
}

// Close closes the LRU cache.
func (s *LRUStorage) Close() error {
	// No resources to release for LRU
	return nil
}
