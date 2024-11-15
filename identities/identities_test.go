package identities

import (
	"orbitdb/go-orbitdb/storage"
	"testing"
)

// Helper function to initialize Identities with a public key provider and storage backend
func setupIdentities(storageBackend storage.Storage) (*Identities, error) {
	return NewIdentities("publickey", storageBackend)
}

func TestNewIdentities(t *testing.T) {
	// Initialize an LRUStorage backend for testing
	lruStorage, err := storage.NewLRUStorage(100)
	if err != nil {
		t.Fatalf("Failed to create LRUStorage: %v", err)
	}

	identities, err := setupIdentities(lruStorage)
	if err != nil {
		t.Fatalf("Expected no error initializing identities, got %v", err)
	}
	if identities == nil {
		t.Fatal("Expected non-nil Identities instance")
	}
}

func TestCreateIdentity(t *testing.T) {
	// Initialize an LRUStorage backend for testing
	lruStorage, err := storage.NewLRUStorage(100)
	if err != nil {
		t.Fatalf("Failed to create LRUStorage: %v", err)
	}

	identities, err := setupIdentities(lruStorage)
	if err != nil {
		t.Fatalf("Error initializing identities: %v", err)
	}

	identity, err := identities.CreateIdentity("test-id")
	if err != nil {
		t.Fatalf("Expected no error creating identity, got %v", err)
	}

	if identity.ID != "test-id" {
		t.Fatalf("Expected identity ID to be 'test-id', got %s", identity.ID)
	}

	if identity.Hash == "" {
		t.Fatal("Expected identity hash to be populated")
	}

	// Verify the identity is stored in the map
	if _, exists := identities.storage[identity.Hash]; !exists {
		t.Fatal("Expected identity to be stored in Identities map")
	}
}

func TestVerifyIdentity(t *testing.T) {
	// Initialize an LRUStorage backend for testing
	lruStorage, err := storage.NewLRUStorage(100)
	if err != nil {
		t.Fatalf("Failed to create LRUStorage: %v", err)
	}

	identities, err := setupIdentities(lruStorage)
	if err != nil {
		t.Fatalf("Error initializing identities: %v", err)
	}

	identity, err := identities.CreateIdentity("test-id")
	if err != nil {
		t.Fatalf("Error creating identity: %v", err)
	}

	// Verify that the created identity is valid
	if !identities.VerifyIdentity(identity) {
		t.Fatal("Expected VerifyIdentity to return true for a valid identity")
	}

	// Tamper with the identity to make it invalid
	identity.ID = "tampered-id"
	if identities.VerifyIdentity(identity) {
		t.Fatal("Expected VerifyIdentity to return false for a tampered identity")
	}
}

func TestSignAndVerify(t *testing.T) {
	// Initialize an LRUStorage backend for testing
	lruStorage, err := storage.NewLRUStorage(100)
	if err != nil {
		t.Fatalf("Failed to create LRUStorage: %v", err)
	}

	identities, err := setupIdentities(lruStorage)
	if err != nil {
		t.Fatalf("Error initializing identities: %v", err)
	}

	identity, err := identities.CreateIdentity("test-id")
	if err != nil {
		t.Fatalf("Error creating identity: %v", err)
	}

	data := []byte("test data")
	// Pass identity ID directly to Sign instead of the Identity struct
	signature, err := identities.Sign(identity.ID, data)
	if err != nil {
		t.Fatalf("Expected no error signing data, got %v", err)
	}

	// Verify the signature with the updated Verify method
	if !identities.Verify(signature, identity, data) {
		t.Fatal("Expected valid signature verification to succeed")
	}

	// Verify with tampered data
	if identities.Verify(signature, identity, []byte("tampered data")) {
		t.Fatal("Expected verification to fail with tampered data")
	}
}
