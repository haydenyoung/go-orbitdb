package keystore

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/json"
	"orbitdb/go-orbitdb/storage"
	"testing"
)

func newTestKeyStore(t *testing.T) *KeyStore {
	// Use an LRUStorage backend for testing.
	lruStorage, err := storage.NewLRUStorage(100)
	if err != nil {
		t.Fatalf("Failed to create LRU storage: %v", err)
	}
	return NewKeyStore(lruStorage)
}

func TestNewKeyStore(t *testing.T) {
	ks := newTestKeyStore(t)
	if ks == nil {
		t.Fatal("Expected KeyStore instance, got nil")
	}
}

func TestCreateKey(t *testing.T) {
	ks := newTestKeyStore(t)
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
	ks := newTestKeyStore(t)
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
	ks := newTestKeyStore(t)
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
	ks := newTestKeyStore(t)
	id := "test-id"

	_, err := ks.CreateKey(id)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Clear all keys
	err = ks.Clear()
	if err != nil {
		t.Fatalf("Expected no error clearing KeyStore, got %v", err)
	}

	if ks.HasKey(id) {
		t.Fatal("Expected HasKey to return false after clearing KeyStore")
	}
}

func TestGetKey(t *testing.T) {
	ks := newTestKeyStore(t)
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

	// Serialize and compare keys
	origBytes, _ := json.Marshal(privateKey)
	retrievedBytes, _ := json.Marshal(retrievedKey)
	if string(origBytes) != string(retrievedBytes) {
		t.Fatal("Expected retrieved key to match the original key")
	}

	// Attempt to retrieve a non-existent key
	_, err = ks.GetKey("nonexistent-id")
	if err == nil {
		t.Fatal("Expected error for non-existent key, got nil")
	}
}

func TestSignMessage(t *testing.T) {
	ks := newTestKeyStore(t)
	id := "test-id"
	data := []byte("test-data")

	// Create a new key and sign a message
	_, err := ks.CreateKey(id)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	signature, err := ks.SignMessage(id, data)
	if err != nil {
		t.Fatalf("Expected no error signing message, got %v", err)
	}

	if len(signature) == 0 {
		t.Fatal("Expected non-empty signature")
	}

	// Attempt to sign with a non-existent key
	_, err = ks.SignMessage("nonexistent-id", data)
	if err == nil {
		t.Fatal("Expected error when signing with non-existent key, got nil")
	}
}

func TestVerifyMessage(t *testing.T) {
	ks := newTestKeyStore(t)
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
	valid, err := ks.VerifyMessage(privateKey.PublicKey, data, signature)
	if err != nil {
		t.Fatalf("Expected no error verifying message, got %v", err)
	}
	if !valid {
		t.Fatal("Expected signature to be valid")
	}

	// Attempt verification with altered data
	valid, err = ks.VerifyMessage(privateKey.PublicKey, []byte("tampered-data"), signature)
	if err != nil {
		t.Fatalf("Expected no error with verification attempt, got %v", err)
	}
	if valid {
		t.Fatal("Expected signature verification to fail with altered data")
	}

	// Attempt verification with an invalid signature format
	invalidSig := "invalid-signature"
	valid, err = ks.VerifyMessage(privateKey.PublicKey, data, invalidSig)
	if err == nil {
		t.Fatal("Expected error with invalid signature format, got nil")
	}
	if valid {
		t.Fatal("Expected invalid signature verification to fail")
	}
}

func TestSerializePrivateKey(t *testing.T) {
	// Generate a test private key
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate private key: %v", err)
	}

	// Serialize the private key
	serializedKey, err := SerializePrivateKey(privateKey)
	if err != nil {
		t.Fatalf("Failed to serialize private key: %v", err)
	}

	// Deserialize the JSON to inspect the fields
	var keyData PrivateKeyData
	if err := json.Unmarshal(serializedKey, &keyData); err != nil {
		t.Fatalf("Failed to unmarshal serialized key: %v", err)
	}

	// Verify fields
	if keyData.Curve != privateKey.Curve.Params().Name {
		t.Errorf("Curve mismatch: expected %s, got %s", privateKey.Curve.Params().Name, keyData.Curve)
	}
	if keyData.X != privateKey.X.Text(16) {
		t.Errorf("X-coordinate mismatch: expected %s, got %s", privateKey.X.Text(16), keyData.X)
	}
	if keyData.Y != privateKey.Y.Text(16) {
		t.Errorf("Y-coordinate mismatch: expected %s, got %s", privateKey.Y.Text(16), keyData.Y)
	}
	if keyData.D != privateKey.D.Text(16) {
		t.Errorf("D value mismatch: expected %s, got %s", privateKey.D.Text(16), keyData.D)
	}
}

func TestDeserializePrivateKey(t *testing.T) {
	// Generate a test private key
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate private key: %v", err)
	}

	// Serialize the private key
	serializedKey, err := SerializePrivateKey(privateKey)
	if err != nil {
		t.Fatalf("Failed to serialize private key: %v", err)
	}

	// Deserialize the private key
	deserializedKey, err := DeserializePrivateKey(serializedKey)
	if err != nil {
		t.Fatalf("Failed to deserialize private key: %v", err)
	}

	// Verify fields of the deserialized key match the original
	if deserializedKey.Curve != privateKey.Curve {
		t.Errorf("Curve mismatch: expected %s, got %s", privateKey.Curve.Params().Name, deserializedKey.Curve.Params().Name)
	}
	if deserializedKey.X.Cmp(privateKey.X) != 0 {
		t.Errorf("X-coordinate mismatch: expected %s, got %s", privateKey.X, deserializedKey.X)
	}
	if deserializedKey.Y.Cmp(privateKey.Y) != 0 {
		t.Errorf("Y-coordinate mismatch: expected %s, got %s", privateKey.Y, deserializedKey.Y)
	}
	if deserializedKey.D.Cmp(privateKey.D) != 0 {
		t.Errorf("D value mismatch: expected %s, got %s", privateKey.D, deserializedKey.D)
	}
}

func TestSerializeAndDeserializePrivateKey(t *testing.T) {
	// Generate a test private key
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate private key: %v", err)
	}

	// Serialize the private key
	serializedKey, err := SerializePrivateKey(privateKey)
	if err != nil {
		t.Fatalf("Failed to serialize private key: %v", err)
	}

	// Deserialize the private key
	deserializedKey, err := DeserializePrivateKey(serializedKey)
	if err != nil {
		t.Fatalf("Failed to deserialize private key: %v", err)
	}

	// Check if the deserialized key is identical to the original
	if deserializedKey.Curve != privateKey.Curve {
		t.Errorf("Curve mismatch: expected %s, got %s", privateKey.Curve.Params().Name, deserializedKey.Curve.Params().Name)
	}
	if deserializedKey.X.Cmp(privateKey.X) != 0 || deserializedKey.Y.Cmp(privateKey.Y) != 0 {
		t.Error("Public key mismatch after deserialization")
	}
	if deserializedKey.D.Cmp(privateKey.D) != 0 {
		t.Error("Private scalar D mismatch after deserialization")
	}
}
