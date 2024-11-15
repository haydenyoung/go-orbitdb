package storage

import (
	"testing"
)

func TestComposedStorage(t *testing.T) {
	// Create two storage backends
	lruStorage, err := NewLRUStorage(2)
	if err != nil {
		t.Fatalf("Failed to create LRUStorage: %v", err)
	}

	memoryStorage, err := NewLRUStorage(2) // Using LRU as a stand-in for another backend
	if err != nil {
		t.Fatalf("Failed to create memory-like LRUStorage: %v", err)
	}

	// Initialize ComposedStorage
	storage, err := NewComposedStorage(lruStorage, memoryStorage)
	if err != nil {
		t.Fatalf("Failed to create ComposedStorage: %v", err)
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

	// Verify propagation to second storage
	err = memoryStorage.Put("key2", []byte("value2"))
	if err != nil {
		t.Fatalf("Failed to put data in memory storage: %v", err)
	}

	value, err = storage.Get("key2")
	if err != nil {
		t.Fatalf("Failed to get propagated data: %v", err)
	}
	if string(value) != "value2" {
		t.Errorf("Expected value2, got %s", value)
	}

	// Test Clear
	err = storage.Clear()
	if err != nil {
		t.Fatalf("Failed to clear ComposedStorage: %v", err)
	}

	_, err = storage.Get("key1")
	if err == nil {
		t.Fatal("Expected key1 to be cleared, but it was found")
	}

	_, err = memoryStorage.Get("key2")
	if err == nil {
		t.Fatal("Expected key2 to be cleared, but it was found")
	}
}
