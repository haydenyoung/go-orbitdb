package oplog

import (
	"fmt"
	"orbitdb/go-orbitdb/identities/identitytypes"
	"orbitdb/go-orbitdb/identities/providers"
	"sync"
	"testing"

	"orbitdb/go-orbitdb/keystore"
	"orbitdb/go-orbitdb/storage"
)

func createIdentityWithProvider(keyStore *keystore.KeyStore, id string) (*identitytypes.Identity, error) {
	// Initialize the PublicKeyProvider with the KeyStore
	provider := providers.NewPublicKeyProvider(keyStore)

	// Use the provider to create an identity
	identity, err := provider.CreateIdentity(id)
	if err != nil {
		return nil, fmt.Errorf("failed to create identity using provider: %w", err)
	}

	return identity, nil
}

func TestLog_AppendAndRetrieve(t *testing.T) {
	storage := storage.NewMemoryStorage()
	keyStore := keystore.NewKeyStore(storage)

	// Create identity using PublicKeyProvider
	identity, err := createIdentityWithProvider(keyStore, "test-identity")
	if err != nil {
		t.Fatalf("Failed to create identity: %v", err)
	}

	log, err := NewLog("test-log", identity, storage, keyStore)
	if err != nil {
		t.Fatalf("Failed to create log: %v", err)
	}

	entry, err := log.Append("Hello, World!")
	if err != nil {
		t.Fatalf("Failed to append entry: %v", err)
	}

	head, err := log.Head()
	if err != nil {
		t.Fatalf("Failed to retrieve log head: %v", err)
	}
	if head.Hash != entry.Hash {
		t.Fatalf("Log head does not match the appended entry")
	}

	retrieved, err := log.Get(entry.Hash)
	if err != nil {
		t.Fatalf("Failed to retrieve entry: %v", err)
	}

	if retrieved.Payload != "Hello, World!" {
		t.Errorf("Expected payload 'Hello, World!', got '%s'", retrieved.Payload)
	}
}

func TestLog_Clear(t *testing.T) {
	storage := storage.NewMemoryStorage()
	keyStore := keystore.NewKeyStore(storage)

	// Create identity using PublicKeyProvider
	identity, err := createIdentityWithProvider(keyStore, "test-identity")
	if err != nil {
		t.Fatalf("Failed to create identity: %v", err)
	}

	log, err := NewLog("test-log", identity, storage, keyStore)
	if err != nil {
		t.Fatalf("Failed to create log: %v", err)
	}

	_, err = log.Append("Entry 1")
	if err != nil {
		t.Fatalf("Failed to append entry: %v", err)
	}

	if err := log.Clear(); err != nil {
		t.Fatalf("Failed to clear log: %v", err)
	}

	if _, err := log.Head(); err == nil {
		t.Fatalf("Expected error when retrieving head of cleared log")
	}

	entries, err := log.Values()
	if err != nil {
		t.Fatalf("Failed to retrieve values after clear: %v", err)
	}

	if len(entries) != 0 {
		t.Errorf("Expected 0 entries after clear, got %d", len(entries))
	}
}

func TestLog_ConcurrentAppend(t *testing.T) {
	storage := storage.NewMemoryStorage()
	keyStore := keystore.NewKeyStore(storage)

	// Create identity using PublicKeyProvider
	identity, err := createIdentityWithProvider(keyStore, "test-identity")
	if err != nil {
		t.Fatalf("Failed to create identity: %v", err)
	}

	log, err := NewLog("test-log", identity, storage, keyStore)
	if err != nil {
		t.Fatalf("Failed to create log: %v", err)
	}

	var wg sync.WaitGroup
	appendCount := 100
	wg.Add(appendCount)

	for i := 0; i < appendCount; i++ {
		go func(i int) {
			defer wg.Done()
			_, err := log.Append(string(rune('A' + i%26)))
			if err != nil {
				t.Errorf("Failed to append entry: %v", err)
			}
		}(i)
	}

	wg.Wait()

	entries, err := log.Values()
	if err != nil {
		t.Fatalf("Failed to retrieve values: %v", err)
	}

	if len(entries) != appendCount {
		t.Errorf("Expected %d entries, got %d", appendCount, len(entries))
	}
}

func TestLog_MultipleAppendsAndRetrieval(t *testing.T) {
	storage := storage.NewMemoryStorage()
	keyStore := keystore.NewKeyStore(storage)

	// Create identity using PublicKeyProvider
	identity, err := createIdentityWithProvider(keyStore, "test-identity")
	if err != nil {
		t.Fatalf("Failed to create identity: %v", err)
	}

	log, err := NewLog("test-log", identity, storage, keyStore)
	if err != nil {
		t.Fatalf("Failed to create log: %v", err)
	}

	payloads := []string{"Entry 1", "Entry 2", "Entry 3"}
	for _, payload := range payloads {
		_, err := log.Append(payload)
		if err != nil {
			t.Fatalf("Failed to append entry: %v", err)
		}
	}

	entries, err := log.Values()
	if err != nil {
		t.Fatalf("Failed to retrieve entries: %v", err)
	}

	if len(entries) != len(payloads) {
		t.Errorf("Expected %d entries, got %d", len(payloads), len(entries))
	}

	for i, entry := range entries {
		if entry.Payload != payloads[i] {
			t.Errorf("Expected payload '%s', got '%s'", payloads[i], entry.Payload)
		}
	}
}

func TestLog_WithCustomKeyStore(t *testing.T) {
	storage := storage.NewMemoryStorage()
	customKeyStore := keystore.NewKeyStore(storage)

	// Create identity using PublicKeyProvider
	identity, err := createIdentityWithProvider(customKeyStore, "custom-identity")
	if err != nil {
		t.Fatalf("Failed to create identity: %v", err)
	}

	log, err := NewLog("test-log", identity, storage, customKeyStore)
	if err != nil {
		t.Fatalf("Failed to create log with custom KeyStore: %v", err)
	}

	entry, err := log.Append("Custom KeyStore Entry")
	if err != nil {
		t.Fatalf("Failed to append entry with custom KeyStore: %v", err)
	}

	head, err := log.Head()
	if err != nil {
		t.Fatalf("Failed to retrieve log head: %v", err)
	}
	if head.Hash != entry.Hash {
		t.Fatalf("Log head does not match the appended entry")
	}

	retrieved, err := log.Get(entry.Hash)
	if err != nil {
		t.Fatalf("Failed to retrieve entry: %v", err)
	}

	if retrieved.Payload != "Custom KeyStore Entry" {
		t.Errorf("Expected payload 'Custom KeyStore Entry', got '%s'", retrieved.Payload)
	}
}
