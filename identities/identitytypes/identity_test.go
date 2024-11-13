package identitytypes

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"github.com/fxamacker/cbor/v2"
	"testing"
)

func TestNewIdentity(t *testing.T) {
	privateKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	identity, err := NewIdentity("test-id", privateKey, "orbitdb")
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
	identity, _ := NewIdentity("test-id", privateKey, "orbitdb")
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
	identity, _ := NewIdentity("test-id", privateKey, "orbitdb")
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

func TestEncodeDecodeIdentity(t *testing.T) {
	privateKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	identity, _ := NewIdentity("test-id", privateKey, "orbitdb")

	// Set up signatures
	identity.Signatures = map[string]string{
		"id":        "signature_for_id",
		"publicKey": "signature_for_public_key",
	}

	// Encode identity
	hash, encodedBytes, err := EncodeIdentity(*identity)
	if err != nil {
		t.Fatalf("Expected no error during encoding, got %v", err)
	}

	// Verify encodedBytes is non-empty
	if len(encodedBytes) == 0 {
		t.Fatal("Expected encoded bytes, got empty slice")
	}

	// Verify hash is non-empty
	if hash == "" {
		t.Fatal("Expected non-empty hash, got empty string")
	}

	// Decode the identity to verify all fields
	decodedIdentity, err := DecodeIdentity(encodedBytes)
	if err != nil {
		t.Fatalf("Failed to decode identity: %v", err)
	}

	// Check ID
	if decodedIdentity.ID != identity.ID {
		t.Fatalf("Expected ID %s, got %s", identity.ID, decodedIdentity.ID)
	}

	// Check PublicKey
	if decodedIdentity.PublicKey != identity.PublicKey {
		t.Fatalf("Expected PublicKey %s, got %s", identity.PublicKey, decodedIdentity.PublicKey)
	}

	// Check Type
	if decodedIdentity.Type != identity.Type {
		t.Fatalf("Expected Type %s, got %s", identity.Type, decodedIdentity.Type)
	}

	// Check Signatures
	if decodedIdentity.Signatures["id"] != identity.Signatures["id"] {
		t.Fatalf("Expected id signature %s, got %s", identity.Signatures["id"], decodedIdentity.Signatures["id"])
	}

	if decodedIdentity.Signatures["publicKey"] != identity.Signatures["publicKey"] {
		t.Fatalf("Expected publicKey signature %s, got %s", identity.Signatures["publicKey"], decodedIdentity.Signatures["publicKey"])
	}
}

func TestDecodeIdentityErrors(t *testing.T) {
	// Test with missing fields
	encodedBytes := []byte{} // empty input
	_, err := DecodeIdentity(encodedBytes)
	if err == nil {
		t.Fatal("Expected error for empty input, got none")
	}

	// Test with missing 'id'
	invalidData := map[string]interface{}{
		"publicKey": "test_public_key",
		"signatures": map[string]string{
			"id":        "signature_for_id",
			"publicKey": "signature_for_public_key",
		},
		"type": "test-type",
	}
	encodedBytes, _ = cbor.Marshal(invalidData)
	_, err = DecodeIdentity(encodedBytes)
	if err == nil || err.Error() != "invalid or missing 'id' field" {
		t.Fatalf("Expected error for missing 'id' field, got %v", err)
	}

	// Test with missing 'publicKey'
	invalidData = map[string]interface{}{
		"id": "test-id",
		"signatures": map[string]string{
			"id":        "signature_for_id",
			"publicKey": "signature_for_public_key",
		},
		"type": "test-type",
	}
	encodedBytes, _ = cbor.Marshal(invalidData)
	_, err = DecodeIdentity(encodedBytes)
	if err == nil || err.Error() != "invalid or missing 'publicKey' field" {
		t.Fatalf("Expected error for missing 'publicKey' field, got %v", err)
	}

	// Test with missing 'signatures'
	invalidData = map[string]interface{}{
		"id":        "test-id",
		"publicKey": "test_public_key",
		"type":      "test-type",
	}
	encodedBytes, _ = cbor.Marshal(invalidData)
	_, err = DecodeIdentity(encodedBytes)
	if err == nil || err.Error() != "invalid or missing 'signatures' field" {
		t.Fatalf("Expected error for missing 'signatures' field, got %v", err)
	}

	// Additional tests for missing or malformed fields within signatures (like 'id' or 'publicKey') can follow a similar pattern.
}

func TestDecodeIdentityInvalidSignatureFormat(t *testing.T) {
	// Simulate malformed signatures with non-string values
	invalidData := map[string]interface{}{
		"id":        "test-id",
		"publicKey": "test_public_key",
		"signatures": map[string]interface{}{
			"id":        123, // Non-string value
			"publicKey": "valid_signature",
		},
		"type": "test-type",
	}

	encodedBytes, _ := cbor.Marshal(invalidData)
	_, err := DecodeIdentity(encodedBytes)
	if err == nil || err.Error() != "invalid signature format" {
		t.Fatalf("Expected error for invalid signature format, got %v", err)
	}
}

func TestNewIdentityErrors(t *testing.T) {
	// Test missing 'id'
	privateKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	_, err := NewIdentity("", privateKey, "orbitdb")
	if err == nil || err.Error() != "identity ID is required" {
		t.Fatalf("Expected error for missing 'id', got %v", err)
	}

	// Test missing 'signatures'
	identity, _ := NewIdentity("test-id", privateKey, "orbitdb")
	identity.Signatures = nil
	if IsIdentity(identity) {
		t.Fatal("Expected false for missing 'signatures', but got true")
	}

	// Test missing 'type'
	_, err = NewIdentity("test-id", privateKey, "")
	if err == nil || err.Error() != "identity type is required" {
		t.Fatalf("Expected error for missing 'type', got %v", err)
	}
}
