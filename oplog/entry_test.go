package oplog

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"fmt"
	"testing"

	"orbitdb/go-orbitdb/identities/identitytypes"
	"orbitdb/go-orbitdb/identities/providers"
)

func generateTestIdentity(t *testing.T) *identitytypes.Identity {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate private key: %v", err)
	}

	provider := providers.NewPublicKeyProvider()
	identity, err := provider.CreateIdentity("test-id", privateKey)
	if err != nil {
		t.Fatalf("Failed to create identity: %v", err)
	}

	return identity
}

func TestNewEntry(t *testing.T) {
	identity := generateTestIdentity(t)
	clock := Clock{ID: "test-clock", Time: 1}
	entry := NewEntry(identity, "entry-id", "payload-data", clock, nil, nil)

	if entry.ID != "entry-id" {
		t.Errorf("Expected entry ID to be 'entry-id', got '%s'", entry.ID)
	}
	if entry.Payload != "payload-data" {
		t.Errorf("Expected payload to be 'payload-data', got '%s'", entry.Payload)
	}
	if entry.Key != identity.PublicKey {
		t.Errorf("Expected entry Key to be '%s', got '%s'", identity.PublicKey, entry.Key)
	}
	if entry.Identity != identity.Hash {
		t.Errorf("Expected entry Identity to be '%s', got '%s'", identity.Hash, entry.Identity)
	}
	if entry.Signature == "" {
		t.Error("Expected entry Signature to be populated, but it was empty")
	}
}

func TestVerifyEntrySignature(t *testing.T) {
	identity := generateTestIdentity(t)
	clock := Clock{ID: "test-clock", Time: 1}
	entry := NewEntry(identity, "entry-id", "payload-data", clock, nil, nil)

	isValid := VerifyEntrySignature(identity, entry)
	if !isValid {
		t.Error("Expected signature to be valid, but verification failed")
	}

	// Modify the payload and check that the signature verification fails
	entry.Payload = "tampered-payload"
	isValid = VerifyEntrySignature(identity, entry)
	if isValid {
		t.Error("Expected signature verification to fail for tampered entry, but it succeeded")
	}
}

func TestIsEntry(t *testing.T) {
	validEntry := Entry{
		ID:      "entry-id",
		Payload: "payload-data",
		Clock:   Clock{ID: "test-clock", Time: 1},
	}

	if !IsEntry(validEntry) {
		t.Error("Expected IsEntry to return true for valid entry")
	}

	invalidEntry := Entry{} // Empty fields
	if IsEntry(invalidEntry) {
		t.Error("Expected IsEntry to return false for invalid entry")
	}
}

func TestIsEqual(t *testing.T) {
	identity := generateTestIdentity(t)
	clock := Clock{ID: "test-clock", Time: 1}

	entry1 := NewEntry(identity, "entry-id", "payload-data", clock, nil, nil)
	entry2 := NewEntry(identity, "entry-id", "payload-data", clock, nil, nil)

	fmt.Printf("Encoded bytes for entry1: %x\n", entry1.Bytes.Bytes())
	fmt.Printf("Encoded bytes for entry2: %x\n", entry2.Bytes.Bytes())

	// Both entries have identical content, so they should have the same serialized bytes
	if !IsEqual(entry1, entry2) {
		t.Error("Expected entries with identical content to be equal")
	}

	// Create an entry with different content and check equality
	entry3 := NewEntry(identity, "entry-id", "different-payload", clock, nil, nil)
	if IsEqual(entry1, entry3) {
		t.Error("Expected entries with different content to not be equal")
	}
}

func TestEncode(t *testing.T) {
	entry := Entry{
		ID:      "entry-id",
		Payload: "payload-data",
		Clock:   Clock{ID: "test-clock", Time: 1},
	}

	encodedEntry := Encode(entry)

	if encodedEntry.CID.String() == "" {
		t.Error("Expected CID to be generated, but it was empty")
	}
	if encodedEntry.Bytes.Len() == 0 {
		t.Error("Expected encoded bytes to be non-empty")
	}
}
