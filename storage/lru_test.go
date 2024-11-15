package storage

import (
	"testing"
)

func TestLRUStorage(t *testing.T) {
	// Initialize LRUStorage
	storage, err := NewLRUStorage(2) // Size 2 for testing eviction
	if err != nil {
		t.Fatalf("Failed to create LRUStorage: %v", err)
	}

	// Test Put and Get
	err = storage.Put("key1", []byte("value1"))
	if err != nil {
		t.Fatalf("Failed to put data: %v", err)
	}

	value, err := storage.Get("key1")
	if err != nil {
		t.Fatalf("Failed to get data: %v", err)
	}
	if string(value) != "value1" {
		t.Errorf("Expected value1, got %s", value)
	}

	// Test eviction when size exceeds limit
	storage.Put("key2", []byte("value2"))
	storage.Put("key3", []byte("value3"))

	_, err = storage.Get("key1") // Should be evicted
	if err == nil {
		t.Fatal("Expected key1 to be evicted, but it was found")
	}

	// Test Clear
	err = storage.Clear()
	if err != nil {
		t.Fatalf("Failed to clear storage: %v", err)
	}

	_, err = storage.Get("key2")
	if err == nil {
		t.Fatal("Expected key2 to be cleared, but it was found")
	}
}
