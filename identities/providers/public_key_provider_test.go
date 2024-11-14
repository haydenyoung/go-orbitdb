package providers

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/hex"
	"math/big"
	"orbitdb/go-orbitdb/keystore"
	"testing"
)

func TestPublicKeyProviderType(t *testing.T) {
	ks := keystore.NewKeyStore()
	provider := NewPublicKeyProvider(ks)
	if provider.Type() != "publickey" {
		t.Fatalf("Expected provider type 'publickey', got %s", provider.Type())
	}
}

func TestCreateIdentity(t *testing.T) {
	ks := keystore.NewKeyStore()
	provider := NewPublicKeyProvider(ks)

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

	// Verify that the ID signature is valid
	publicKeyBytes, err := hex.DecodeString(identity.PublicKey)
	if err != nil {
		t.Fatalf("Error decoding public key: %v", err)
	}

	pubKey := ecdsa.PublicKey{
		Curve: elliptic.P256(),
		X:     new(big.Int).SetBytes(publicKeyBytes[:len(publicKeyBytes)/2]),
		Y:     new(big.Int).SetBytes(publicKeyBytes[len(publicKeyBytes)/2:]),
	}

	idVerified, err := keystore.VerifyMessage(pubKey, []byte(identity.ID), identity.Signatures["id"])
	if err != nil || !idVerified {
		t.Fatal("Expected ID signature to be valid")
	}

	publicKeyVerified, err := keystore.VerifyMessage(pubKey, []byte(identity.PublicKey), identity.Signatures["publicKey"])
	if err != nil || !publicKeyVerified {
		t.Fatal("Expected public key signature to be valid")
	}
}

func TestVerifyIdentity(t *testing.T) {
	ks := keystore.NewKeyStore()
	provider := NewPublicKeyProvider(ks)

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
