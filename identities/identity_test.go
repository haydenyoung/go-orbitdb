package identities

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"math/big"
	"testing"
)

// TestIdentityKeyGeneration checks if identity creation generates valid keys and ID.
func TestIdentityKeyGeneration(t *testing.T) {
	// Generate a new identity
	identity, err := NewIdentity()
	if err != nil {
		t.Fatalf("Failed to create identity: %v", err)
	}

	// Check that the public key X and Y coordinates are non-nil
	if identity.PublicKey.X == nil || identity.PublicKey.Y == nil {
		t.Error("Public key coordinates should not be nil")
	}

	// Check that the private key D value is non-nil
	if identity.PrivateKey.D == nil {
		t.Error("Private key D value should not be nil")
	}

	// Ensure identity ID is non-empty
	if identity.ID == "" {
		t.Error("Expected non-empty ID for identity")
	}
}

func TestIdentityCreation(t *testing.T) {
	id, err := NewIdentity()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if id.ID == "" {
		t.Errorf("Expected non-empty ID, got empty string")
	}
	if id.Type != "publickey" {
		t.Errorf("Expected type 'publickey', got %v", id.Type)
	}
	if id.PublicKeyHex() == "" {
		t.Errorf("Expected non-empty public key hex, got empty string")
	}
}

func TestIdentityExpectedHashAndBytes(t *testing.T) {
	// Sample expected hash and bytes for the identity
	expectedHash := "zdpuArx43BnXdDff5rjrGLYrxUomxNroc2uaocTgcWK76UfQT"
	expectedBytes := []byte{164, 98, 105, 100, 120, 39, 48, 120, 48, 49, 50, 51, 52, 53, 54, 55, 56, 57, 48, 97, 98, 99, 100, 101, 102, 103, 104, 105, 106, 107, 108, 109, 110, 111, 112, 113, 114, 115, 116, 117, 118, 119, 120, 121, 122, 100, 116, 121, 112, 101, 103, 111, 114, 98, 105, 116, 100, 98, 105, 112, 117, 98, 108, 105, 99, 75, 101, 121, 104, 60, 112, 117, 98, 107, 101, 121, 62, 106, 115, 105, 103, 110, 97, 116, 117, 114, 101, 115, 162, 98, 105, 100, 114, 115, 105, 103, 110, 97, 116, 117, 114, 101, 32, 102, 111, 114, 32, 60, 105, 100, 62, 105, 112, 117, 98, 108, 105, 99, 75, 101, 121, 120, 39, 115, 105, 103, 110, 97, 116, 117, 114, 101, 32, 102, 111, 114, 32, 60, 112, 117, 98, 108, 105, 99, 75, 101, 121, 32, 43, 32, 105, 100, 83, 105, 103, 110, 97, 116, 117, 114, 101, 62}

	id, _ := NewIdentity()

	// Generate hash for the public key
	pubKeyBytes := append(id.PublicKey.X.Bytes(), id.PublicKey.Y.Bytes()...)
	hash := sha256.Sum256(pubKeyBytes)
	hashHex := hex.EncodeToString(hash[:])

	// Check if the generated hash matches the expected hash
	if hashHex != expectedHash {
		t.Errorf("Expected hash %v, got %v", expectedHash, hashHex)
	}

	// Simulate encoding the identity into bytes (using JSON as a placeholder)
	actualBytes, _ := json.Marshal(id)
	if !bytes.Equal(actualBytes, expectedBytes) {
		t.Errorf("Expected bytes %v, got %v", expectedBytes, actualBytes)
	}
}

func TestIdentityEquality(t *testing.T) {
	id1, _ := NewIdentity()
	id2 := &Identity{
		ID:         id1.ID,
		PublicKey:  id1.PublicKey,
		PrivateKey: id1.PrivateKey,
		Type:       id1.Type,
		Signatures: id1.Signatures,
	}

	// Function to compare two identities
	isEqual := func(a, b *Identity) bool {
		return a.ID == b.ID && a.PublicKeyHex() == b.PublicKeyHex() && a.Type == b.Type
	}

	// Test equal identities
	if !isEqual(id1, id2) {
		t.Error("Expected identities to be equal")
	}

	// Modify id2 and check if they are not equal
	id2.ID = "different-id"
	if isEqual(id1, id2) {
		t.Error("Expected identities to be not equal")
	}
}

// TestPublicKeyHex verifies that the PublicKeyHex method correctly returns the hex representation of the public key.
func TestPublicKeyHex(t *testing.T) {
	// Generate a new identity
	identity, err := NewIdentity()
	if err != nil {
		t.Fatalf("Failed to create identity: %v", err)
	}

	// Get the hex-encoded public key
	pubKeyHex := identity.PublicKeyHex()

	// Decode the hex string to verify it’s valid hex
	pubKeyBytes, err := hex.DecodeString(pubKeyHex)
	if err != nil {
		t.Fatalf("Failed to decode public key hex: %v", err)
	}

	// Check that the decoded bytes match the original public key coordinates
	expectedPubKeyBytes := append(identity.PublicKey.X.Bytes(), identity.PublicKey.Y.Bytes()...)
	if !bytes.Equal(pubKeyBytes, expectedPubKeyBytes) {
		t.Error("Decoded public key bytes do not match original public key bytes")
	}

	t.Logf("Public Key (Hex): %s", pubKeyHex)
}

func TestSignAndVerify(t *testing.T) {
	// Generate a new identity
	identity, err := NewIdentity()
	if err != nil {
		t.Fatalf("Failed to create identity: %v", err)
	}

	// Data to sign
	data := []byte("test data")
	hash := sha256.Sum256(data)

	// Sign the data
	signature, err := identity.Sign(data)
	if err != nil {
		t.Fatalf("Failed to sign data: %v", err)
	}

	// Verify the signature manually by decoding it and verifying using ecdsa.Verify
	r := new(big.Int)
	s := new(big.Int)
	sigLen := len(signature) / 2
	r.SetString(signature[:sigLen], 16)
	s.SetString(signature[sigLen:], 16)

	// Perform ECDSA verification
	valid := ecdsa.Verify(&identity.PublicKey, hash[:], r, s)
	if !valid {
		t.Error("Signature verification failed")
	}

	t.Log("Signature successfully verified")
}

// TestInvalidSignature ensures that a modified signature does not verify correctly.
func TestInvalidSignature(t *testing.T) {
	// Generate a new identity
	identity, err := NewIdentity()
	if err != nil {
		t.Fatalf("Failed to create identity: %v", err)
	}

	// Data to sign
	data := []byte("test data")

	// Sign the data
	signature, err := identity.Sign(data)
	if err != nil {
		t.Fatalf("Failed to sign data: %v", err)
	}

	// Modify the signature slightly
	alteredSignature := signature[:len(signature)-1] + "0"

	// Manually verify that the altered signature fails
	hash := sha256.Sum256(data)
	r := new(big.Int)
	s := new(big.Int)
	sigLen := len(alteredSignature) / 2
	r.SetString(alteredSignature[:sigLen], 16)
	s.SetString(alteredSignature[sigLen:], 16)

	// Ensure the altered signature does not pass
	valid := ecdsa.Verify(&identity.PublicKey, hash[:], r, s)
	if valid {
		t.Error("Expected signature verification to fail for altered data, but it succeeded")
	}
}
