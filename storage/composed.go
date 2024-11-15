package storage

import (
	"errors"
)

// ComposedStorage implements the Storage interface and manages multiple backends.
type ComposedStorage struct {
	storages []Storage
}

// NewComposedStorage initializes a ComposedStorage instance with multiple backends.
func NewComposedStorage(storages ...Storage) (*ComposedStorage, error) {
	if len(storages) < 2 {
		return nil, errors.New("at least two storage backends are required")
	}
	return &ComposedStorage{storages: storages}, nil
}

// Put stores data in all configured storages.
func (cs *ComposedStorage) Put(key string, value []byte) error {
	for _, storage := range cs.storages {
		if err := storage.Put(key, value); err != nil {
			return err
		}
	}
	return nil
}

// Get retrieves data from the first storage that has the key.
// If the key is found in a fallback storage, it is propagated to the earlier storages.
func (cs *ComposedStorage) Get(key string) ([]byte, error) {
	for i, storage := range cs.storages {
		value, err := storage.Get(key)
		if err == nil {
			// Propagate to earlier storages if retrieved from a fallback storage.
			for j := 0; j < i; j++ {
				_ = cs.storages[j].Put(key, value) // Ignore errors during propagation.
			}
			return value, nil
		}
	}
	return nil, errors.New("key not found")
}

// Delete removes data from all storages.
func (cs *ComposedStorage) Delete(key string) error {
	for _, storage := range cs.storages {
		if err := storage.Delete(key); err != nil {
			return err
		}
	}
	return nil
}

// Iterator combines iterators from all storages, ensuring unique keys.
func (cs *ComposedStorage) Iterator() (<-chan [2]string, error) {
	ch := make(chan [2]string)
	seen := make(map[string]bool)

	go func() {
		defer close(ch)
		for _, storage := range cs.storages {
			iter, err := storage.Iterator()
			if err != nil {
				continue // Skip problematic storage during iteration.
			}

			for kv := range iter {
				if !seen[kv[0]] {
					seen[kv[0]] = true
					ch <- kv
				}
			}
		}
	}()

	return ch, nil
}

// Merge merges data from another storage into all composed storages.
func (cs *ComposedStorage) Merge(other Storage) error {
	iter, err := other.Iterator()
	if err != nil {
		return err
	}

	for kv := range iter {
		if err := cs.Put(kv[0], []byte(kv[1])); err != nil {
			return err
		}
	}

	return nil
}

// Clear removes all data from all storages.
func (cs *ComposedStorage) Clear() error {
	for _, storage := range cs.storages {
		if err := storage.Clear(); err != nil {
			return err
		}
	}
	return nil
}

// Close closes all storage backends.
func (cs *ComposedStorage) Close() error {
	for _, storage := range cs.storages {
		if err := storage.Close(); err != nil {
			return err
		}
	}
	return nil
}
