package identities

import (
	"testing"
)

func TestNewIdentities(t *testing.T) {
	ids, err := NewIdentities("publickey")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if ids == nil {
		t.Fatal("Expected identities instance, got nil")
	}

	if ids.provider.Type() != "publickey" {
		t.Fatalf("Expected provider type 'publickey', got %s", ids.provider.Type())
	}
}

func TestCreateIdentity(t *testing.T) {
	ids, _ := NewIdentities("publickey")
	identity, err := ids.CreateIdentity("test-id")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if identity == nil {
		t.Fatal("Expected identity, got nil")
	}

	if identity.ID != "test-id" {
		t.Fatalf("Expected ID 'test-id', got %s", identity.ID)
	}

	storedIdentity := ids.storage[identity.Hash]
	if storedIdentity == nil {
		t.Fatal("Expected identity to be stored in identities storage, but it wasn't")
	}
}

func TestVerifyIdentity(t *testing.T) {
	ids, _ := NewIdentities("publickey")
	identity, _ := ids.CreateIdentity("test-id")
	data := []byte("test data")
	signature, _ := identity.Sign(data)

	// Test with correct data and signature
	if !ids.VerifyIdentity(identity, signature, data) {
		t.Fatal("Expected identity verification to succeed with valid data, but it failed")
	}

	// Test with tampered data
	invalidData := []byte("tampered data")
	if ids.VerifyIdentity(identity, signature, invalidData) {
		t.Fatal("Expected identity verification to fail with tampered data, but it succeeded")
	}

	// Test with an invalid signature
	invalidSignature := "abcdef" // Clearly invalid signature
	if ids.VerifyIdentity(identity, invalidSignature, data) {
		t.Fatal("Expected identity verification to fail with an invalid signature, but it succeeded")
	}

	// Test with a different identity (not matching the signature)
	otherIdentity, _ := ids.CreateIdentity("other-id")
	// FIXME it fails here because of same hardcoded priv-key
	if ids.VerifyIdentity(otherIdentity, signature, data) {
		t.Fatal("Expected verification to fail with mismatched identity and signature, but it succeeded")
	}
}
