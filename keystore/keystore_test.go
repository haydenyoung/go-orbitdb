package keystore

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"testing"
)

func TestNewKeyStore(t *testing.T) {
	ks := NewKeyStore()
	if ks == nil {
		t.Fatal("Expected KeyStore instance, got nil")
	}
}

func TestCreateKey(t *testing.T) {
	ks := NewKeyStore()
	id := "test-id"

	_, err := ks.CreateKey(id)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Attempt to create a key with the same ID again
	_, err = ks.CreateKey(id)
	if err == nil {
		t.Fatal("Expected error when creating duplicate key, got nil")
	}
}

func TestHasKey(t *testing.T) {
	ks := NewKeyStore()
	id := "test-id"

	if ks.HasKey(id) {
		t.Fatal("Expected HasKey to return false for nonexistent key")
	}

	_, err := ks.CreateKey(id)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if !ks.HasKey(id) {
		t.Fatal("Expected HasKey to return true for existing key")
	}
}

func TestAddKey(t *testing.T) {
	ks := NewKeyStore()
	id := "test-id"

	// Generate a new ECDSA key pair
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("Error generating test private key: %v", err)
	}

	// Add the key to the KeyStore
	err = ks.AddKey(id, privateKey)
	if err != nil {
		t.Fatalf("Expected no error adding key, got %v", err)
	}

	// Attempt to add the same key again
	err = ks.AddKey(id, privateKey)
	if err == nil {
		t.Fatal("Expected error when adding duplicate key, got nil")
	}
}

func TestClear(t *testing.T) {
	ks := NewKeyStore()
	id := "test-id"

	_, err := ks.CreateKey(id)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Clear all keys
	ks.Clear()

	if ks.HasKey(id) {
		t.Fatal("Expected HasKey to return false after clearing KeyStore")
	}
}

func TestGetKey(t *testing.T) {
	ks := NewKeyStore()
	id := "test-id"

	// Create a new key
	privateKey, err := ks.CreateKey(id)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Retrieve the key
	retrievedKey, err := ks.GetKey(id)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if retrievedKey != privateKey {
		t.Fatal("Expected retrieved key to match the original key")
	}

	// Attempt to retrieve a non-existent key
	_, err = ks.GetKey("nonexistent-id")
	if err == nil {
		t.Fatal("Expected error for non-existent key, got nil")
	}
}

func TestSignMessage(t *testing.T) {
	ks := NewKeyStore()
	id := "test-id"
	data := []byte("test-data")

	// Create a new key and sign a message
	_, err := ks.CreateKey(id)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	_, err = ks.SignMessage(id, data)
	if err != nil {
		t.Fatalf("Expected no error signing message, got %v", err)
	}

	// Attempt to sign with a non-existent key
	_, err = ks.SignMessage("nonexistent-id", data)
	if err == nil {
		t.Fatal("Expected error when signing with non-existent key, got nil")
	}
}

func TestVerifyMessage(t *testing.T) {
	ks := NewKeyStore()
	id := "test-id"
	data := []byte("test-data")

	// Create a new key and sign a message
	privateKey, err := ks.CreateKey(id)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	signature, err := ks.SignMessage(id, data)
	if err != nil {
		t.Fatalf("Expected no error signing message, got %v", err)
	}

	// Verify the message using the public key
	valid, err := VerifyMessage(privateKey.PublicKey, data, signature)
	if err != nil {
		t.Fatalf("Expected no error verifying message, got %v", err)
	}
	if !valid {
		t.Fatal("Expected signature to be valid")
	}

	// Attempt verification with altered data
	valid, err = VerifyMessage(privateKey.PublicKey, []byte("tampered-data"), signature)
	if err != nil {
		t.Fatalf("Expected no error with verification attempt, got %v", err)
	}
	if valid {
		t.Fatal("Expected signature verification to fail with altered data")
	}

	// Attempt verification with an invalid signature format
	invalidSig := "invalid-signature"
	valid, err = VerifyMessage(privateKey.PublicKey, data, invalidSig)
	if err == nil {
		t.Fatal("Expected error with invalid signature format, got nil")
	}
	if valid {
		t.Fatal("Expected invalid signature verification to fail")
	}
}
