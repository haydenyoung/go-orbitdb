package oplog

import (
	"bytes"
	"encoding/hex"
	"testing"
)

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
}

func TestPublicKeyHex(t *testing.T) {
	// Generate a new identity
	identity, err := NewIdentity()
	if err != nil {
		t.Fatalf("Failed to create identity: %v", err)
	}

	// Get the public key as a hex string
	pubKeyHex := identity.PublicKeyHex()

	// Decode the hex string to verify it’s valid hex
	pubKeyBytes, err := hex.DecodeString(pubKeyHex)
	if err != nil {
		t.Fatalf("Failed to decode public key hex: %v", err)
	}

	// Check that the decoded bytes match the original public key
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

	// Sign the data
	signature, err := identity.Sign(data)
	if err != nil {
		t.Fatalf("Failed to sign data: %v", err)
	}

	// Verify the signature
	valid, err := identity.VerifySignature(data, signature)
	if err != nil {
		t.Fatalf("Failed to verify signature: %v", err)
	}
	if !valid {
		t.Error("Signature verification failed")
	}

	t.Log("Signature successfully verified")
}

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

	// Attempt to verify with altered data
	alteredData := []byte("altered test data")
	valid, err := identity.VerifySignature(alteredData, signature)
	if err != nil {
		t.Fatalf("Error during signature verification: %v", err)
	}
	if valid {
		t.Error("Expected signature verification to fail for altered data, but it succeeded")
	}
}
