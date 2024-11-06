package oplog

import "testing"

func TestIdentity(t *testing.T) {
	// Create a new identity
	identity, err := NewIdentity()
	if err != nil {
		t.Fatalf("Failed to create identity: %v", err)
	}

	// Sign some data
	data := []byte("test data")
	signature, err := identity.Sign(data)
	if err != nil {
		t.Fatalf("Failed to sign data: %v", err)
	}

	// Verify the signature
	valid, err := identity.VerifySignature(data, signature)
	if err != nil || !valid {
		t.Fatalf("Signature verification failed: %v", err)
	}
}
