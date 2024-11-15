package storage

import (
	"testing"
)

func TestMemoryStorage_PutAndGet(t *testing.T) {
	// Initialize MemoryStorage
	memStorage := NewMemoryStorage()

	// Test putting and getting data
	err := memStorage.Put("key1", []byte("value1"))
	if err != nil {
		t.Fatalf("Failed to put data: %v", err)
	}

	value, err := memStorage.Get("key1")
	if err != nil {
		t.Fatalf("Failed to get data: %v", err)
	}
	if string(value) != "value1" {
		t.Errorf("Expected value1, got %s", value)
	}

	// Test getting non-existent key
	_, err = memStorage.Get("nonexistent")
	if err == nil {
		t.Fatal("Expected error for non-existent key, got nil")
	}
}

func TestMemoryStorage_Delete(t *testing.T) {
	// Initialize MemoryStorage
	memStorage := NewMemoryStorage()

	// Add and delete a key
	err := memStorage.Put("key1", []byte("value1"))
	if err != nil {
		t.Fatalf("Failed to put data: %v", err)
	}

	err = memStorage.Delete("key1")
	if err != nil {
		t.Fatalf("Failed to delete data: %v", err)
	}

	_, err = memStorage.Get("key1")
	if err == nil {
		t.Fatal("Expected error for deleted key, got nil")
	}
}

func TestMemoryStorage_Clear(t *testing.T) {
	// Initialize MemoryStorage
	memStorage := NewMemoryStorage()

	// Add some keys
	memStorage.Put("key1", []byte("value1"))
	memStorage.Put("key2", []byte("value2"))

	// Clear the storage
	err := memStorage.Clear()
	if err != nil {
		t.Fatalf("Failed to clear storage: %v", err)
	}

	// Verify storage is empty
	_, err = memStorage.Get("key1")
	if err == nil {
		t.Fatal("Expected error for cleared key, got nil")
	}

	_, err = memStorage.Get("key2")
	if err == nil {
		t.Fatal("Expected error for cleared key, got nil")
	}
}

func TestMemoryStorage_Iterator(t *testing.T) {
	// Initialize MemoryStorage
	memStorage := NewMemoryStorage()

	// Add some keys
	memStorage.Put("key1", []byte("value1"))
	memStorage.Put("key2", []byte("value2"))

	// Iterate over key-value pairs
	iter, err := memStorage.Iterator()
	if err != nil {
		t.Fatalf("Failed to get iterator: %v", err)
	}

	expected := map[string]string{
		"key1": "value1",
		"key2": "value2",
	}

	for kv := range iter {
		key, value := kv[0], kv[1]
		if expected[key] != value {
			t.Errorf("Unexpected key-value pair: %s=%s", key, value)
		}
		delete(expected, key)
	}

	// Ensure all keys were iterated
	if len(expected) != 0 {
		t.Errorf("Some keys were not iterated: %v", expected)
	}
}

func TestMemoryStorage_Merge(t *testing.T) {
	// Initialize two MemoryStorage instances
	memStorage1 := NewMemoryStorage()
	memStorage2 := NewMemoryStorage()

	// Add keys to both storages
	memStorage1.Put("key1", []byte("value1"))
	memStorage2.Put("key2", []byte("value2"))

	// Merge memStorage2 into memStorage1
	err := memStorage1.Merge(memStorage2)
	if err != nil {
		t.Fatalf("Failed to merge storage: %v", err)
	}

	// Verify merged keys
	value, err := memStorage1.Get("key1")
	if err != nil || string(value) != "value1" {
		t.Errorf("Expected key1=value1, got %s", value)
	}

	value, err = memStorage1.Get("key2")
	if err != nil || string(value) != "value2" {
		t.Errorf("Expected key2=value2, got %s", value)
	}
}
