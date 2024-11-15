package storage

import (
	"os"
	"testing"
)

func TestLevelStorage(t *testing.T) {
	// Create a temporary directory for LevelStorage
	path := "./test-leveldb"
	defer os.RemoveAll(path) // Clean up after test

	// Initialize LevelStorage
	storage, err := NewLevelStorage(path)
	if err != nil {
		t.Fatalf("Failed to create LevelStorage: %v", err)
	}
	defer storage.Close()

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

	// Test non-existent key
	_, err = storage.Get("nonexistent")
	if err == nil {
		t.Fatal("Expected error for non-existent key, got nil")
	}

	// Test Delete
	err = storage.Delete("key1")
	if err != nil {
		t.Fatalf("Failed to delete data: %v", err)
	}

	_, err = storage.Get("key1")
	if err == nil {
		t.Fatal("Expected error for deleted key, got nil")
	}

	// Test Clear
	err = storage.Put("key2", []byte("value2"))
	if err != nil {
		t.Fatalf("Failed to put data: %v", err)
	}

	err = storage.Clear()
	if err != nil {
		t.Fatalf("Failed to clear storage: %v", err)
	}

	_, err = storage.Get("key2")
	if err == nil {
		t.Fatal("Expected error for cleared key, got nil")
	}
}
