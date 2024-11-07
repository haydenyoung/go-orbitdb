package identities

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/hex"
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
