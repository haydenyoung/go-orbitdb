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

	if identity.Type != "publickey" {
		t.Fatalf("Expected identity type 'publickey', got %s", identity.Type)
	}
}

func TestVerifyIdentity(t *testing.T) {
	provider := NewPublicKeyProvider()
	identity, err := provider.CreateIdentity("test-id")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Test that a valid identity passes verification
	valid, err := provider.VerifyIdentity(identity)
	if err != nil {
		t.Fatalf("Expected no error during identity verification, got %v", err)
	}
	if !valid {
		t.Fatal("Expected VerifyIdentity to return true for a valid identity")
	}

	// Modify identity to make it invalid
	identity.ID = "tampered-id"
	valid, err = provider.VerifyIdentity(identity)
	if valid || err == nil {
		t.Fatal("Expected VerifyIdentity to return false for a tampered identity")
	}
}

func TestSignAndVerify(t *testing.T) {
	provider := NewPublicKeyProvider()
	identity, err := provider.CreateIdentity("test-id")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	data := "test-data"
	signature, err := provider.Sign(data, identity)
	if err != nil {
		t.Fatalf("Expected no error signing data, got %v", err)
	}

	// Verify that the signature is valid
	if !provider.Verify(identity, signature, []byte(data)) {
		t.Fatal("Expected valid signature verification to return true")
	}

	// Test with altered data
	if provider.Verify(identity, signature, []byte("tampered-data")) {
		t.Fatal("Expected verification to fail with altered data")
	}
}
