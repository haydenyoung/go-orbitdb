package identitytypes

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"testing"
)

func TestNewIdentity(t *testing.T) {
	privateKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	identity, err := NewIdentity("test-id", privateKey)
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

func TestSign(t *testing.T) {
	privateKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	identity, _ := NewIdentity("test-id", privateKey)
	data := []byte("data to sign")

	signature, err := identity.Sign(data)
	if err != nil {
		t.Fatalf("Expected no error during signing, got %v", err)
	}

	if signature == "" {
		t.Fatal("Expected signature, got empty string")
	}
}

func TestVerify(t *testing.T) {
	privateKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	identity, _ := NewIdentity("test-id", privateKey)
	data := []byte("data to sign")
	signature, _ := identity.Sign(data)

	// Test valid signature and data
	if !identity.Verify(signature, data) {
		t.Fatal("Expected verification to succeed with valid data, but it failed")
	}

	// Test with tampered data
	invalidData := []byte("tampered data")
	if identity.Verify(signature, invalidData) {
		t.Fatal("Expected verification to fail with tampered data, but it succeeded")
	}

	// Test with a clearly invalid signature
	invalidSignature := "invalidsignature"
	if identity.Verify(invalidSignature, data) {
		t.Fatal("Expected verification to fail with an invalid signature, but it succeeded")
	}

	// Test with an incorrect signature (generate a new signature for different data)
	otherData := []byte("different data")
	otherSignature, _ := identity.Sign(otherData)
	if identity.Verify(otherSignature, data) {
		t.Fatal("Expected verification to fail with a mismatched signature, but it succeeded")
	}
}
