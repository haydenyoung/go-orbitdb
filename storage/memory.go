package storage

import (
	"errors"
	"sync"
)

// MemoryStorage stores data in memory
type MemoryStorage struct {
	memory map[string][]byte
	mu     sync.RWMutex
}

// NewMemoryStorage creates a new instance of MemoryStorage
func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		memory: make(map[string][]byte),
	}
}

// Put stores data in memory
func (ms *MemoryStorage) Put(hash string, data []byte) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.memory[hash] = data
	return nil
}

// Get retrieves data from memory
func (ms *MemoryStorage) Get(hash string) ([]byte, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()
	value, exists := ms.memory[hash]
	if !exists {
		return nil, errors.New("key not found")
	}
	return value, nil
}

// Delete removes data from memory
func (ms *MemoryStorage) Delete(hash string) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	delete(ms.memory, hash)
	return nil
}

// Iterator iterates over all stored key-value pairs
func (ms *MemoryStorage) Iterator() (<-chan [2]string, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()
	ch := make(chan [2]string)

	go func() {
		defer close(ch)
		for key, value := range ms.memory {
			ch <- [2]string{key, string(value)}
		}
	}()

	return ch, nil
}

// Merge merges data from another storage instance into memory
func (ms *MemoryStorage) Merge(other Storage) error {
	iter, err := other.Iterator()
	if err != nil {
		return err
	}

	for kv := range iter {
		ms.Put(kv[0], []byte(kv[1]))
	}

	return nil
}

// Clear clears all data in memory
func (ms *MemoryStorage) Clear() error {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.memory = make(map[string][]byte)
	return nil
}

// Close closes the memory storage (noop for in-memory)
func (ms *MemoryStorage) Close() error {
	return nil
}
