package providers

import (
	"testing"
)

func TestPublicKeyProviderType(t *testing.T) {
	provider := NewPublicKeyProvider()
	if provider.Type() != "publickey" {
		t.Fatalf("Expected provider type 'publickey', got %s", provider.Type())
	}
}

func TestCreateIdentity(t *testing.T) {
	provider := NewPublicKeyProvider()
	identity, err := provider.CreateIdentity("test-id")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if identity.ID != "test-id" {
		t.Fatalf("Expected ID 'test-id', got %s", identity.ID)
	}

	if identity.PublicKey == "" {
		t.Fatal("Expected public key to be set, got empty string")
	}

	if identity.Hash == "" {
		t.Fatal("Expected identity hash to be set, got empty string")
	}
}

func TestVerifyIdentity(t *testing.T) {
	provider := NewPublicKeyProvider()
	identity, _ := provider.CreateIdentity("test-id")
	data := []byte("data to verify")
	signature, _ := identity.Sign(data)

	// Test valid verification
	if !provider.VerifyIdentity(identity, signature, data) {
		t.Fatal("Expected verification to succeed with valid data, but it failed")
	}

	// Test with tampered data
	invalidData := []byte("tampered data")
	if provider.VerifyIdentity(identity, signature, invalidData) {
		t.Fatal("Expected verification to fail with tampered data, but it succeeded")
	}

	// Test with a clearly invalid signature
	invalidSignature := "abcdef"
	if provider.VerifyIdentity(identity, invalidSignature, data) {
		t.Fatal("Expected verification to fail with an invalid signature, but it succeeded")
	}

	// Test with a mismatched identity and signature
	otherIdentity, _ := provider.CreateIdentity("other-id")
	// FIXME it fails here because of same hardcoded priv-key
	if provider.VerifyIdentity(otherIdentity, signature, data) {
		t.Fatal("Expected verification to fail with a mismatched identity and signature, but it succeeded")
	}
}
