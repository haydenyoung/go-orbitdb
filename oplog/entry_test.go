package oplog

import (
	"bytes"
	"orbitdb/go-orbitdb/keystore"
	"orbitdb/go-orbitdb/storage"
	"testing"

	"orbitdb/go-orbitdb/identities/identitytypes"
	"orbitdb/go-orbitdb/identities/providers"
)

func setupTestKeyStoreAndIdentity(t *testing.T) (*keystore.KeyStore, *identitytypes.Identity) {
	// Use LRUStorage as the storage backend for testing
	lruStorage, err := storage.NewLRUStorage(100)
	if err != nil {
		panic(err) // Ensure setup failure is immediately apparent
	}
	ks := keystore.NewKeyStore(lruStorage)
	provider := providers.NewPublicKeyProvider(ks)
	identity, err := provider.CreateIdentity("test-ID")
	if err != nil {
		t.Fatalf("Failed to create identity: %v", err)
	}
	return ks, identity
}

func TestNewEntry(t *testing.T) {
	ks, identity := setupTestKeyStoreAndIdentity(t)
	clock := Clock{ID: "test-clock", Time: 1}
	entry := NewEntry(ks, identity, "entry-ID", "payload-data", clock, nil, nil)

	if entry.ID != "entry-ID" {
		t.Errorf("Expected entry ID to be 'entry-ID', got '%s'", entry.ID)
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
	ks, identity := setupTestKeyStoreAndIdentity(t)
	clock := Clock{ID: "test-clock", Time: 1}
	entry := NewEntry(ks, identity, "entry-ID", "payload-data", clock, nil, nil)

	isValid := VerifyEntrySignature(ks, entry)
	if !isValid {
		t.Error("Expected signature to be valid, but verification failed")
	}

	// Modify the payload and check that the signature verification fails
	entry.Entry.Payload = "tampered-payload"
	isValid = VerifyEntrySignature(ks, entry)
	if isValid {
		t.Error("Expected signature verification to fail for tampered entry, but it succeeded")
	}
}

func TestIsEntry(t *testing.T) {
	validEntry := Entry{
		ID:      "entry-ID",
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
	ks, identity := setupTestKeyStoreAndIdentity(t)
	clock := Clock{ID: "test-clock", Time: 1}

	entry1 := NewEntry(ks, identity, "entry-ID", "payload-data", clock, nil, nil)
	entry2 := NewEntry(ks, identity, "entry-ID", "payload-data", clock, nil, nil)

	// Both Entries have identical content, so they should have the same serialized bytes
	if !IsEqual(entry1, entry2) {
		t.Error("Expected Entries with identical content to be equal")
	}

	// Create an entry with different content and check equality
	entry3 := NewEntry(ks, identity, "entry-ID", "different-payload", clock, nil, nil)
	if IsEqual(entry1, entry3) {
		t.Error("Expected Entries with different content to not be equal")
	}
}

func TestEncode(t *testing.T) {
	entry := Entry{
		ID:      "entry-ID",
		Payload: "payload-data",
		Clock:   Clock{ID: "test-clock", Time: 1},
	}

	encodedEntry := Encode(entry)

	if encodedEntry.CID.String() == "" {
		t.Error("Expected CID to be generated, but it was empty")
	}
	if len(encodedEntry.Bytes) == 0 {
		t.Error("Expected encoded bytes to be non-empty")
	}
}

func TestDecode(t *testing.T) {
	// Create a sample entry
	entry := Entry{
		ID:        "entry-ID",
		Payload:   "payload-data",
		Clock:     Clock{ID: "test-clock", Time: 1},
		V:         2,
		Key:       "test-key",
		Identity:  "test-identity",
		Signature: "test-signature",
	}

	// Encode the entry
	encodedEntry := Encode(entry)

	// Decode the encoded bytes back into an EncodedEntry
	decodedEntry, err := Decode(encodedEntry.Bytes)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	// Verify that the decoded entry matches the original entry
	if decodedEntry.Entry.ID != entry.ID {
		t.Errorf("Expected ID %s, got %s", entry.ID, decodedEntry.Entry.ID)
	}
	if decodedEntry.Entry.Payload != entry.Payload {
		t.Errorf("Expected Payload %s, got %s", entry.Payload, decodedEntry.Entry.Payload)
	}
	if decodedEntry.Entry.Clock.ID != entry.Clock.ID {
		t.Errorf("Expected Clock ID %s, got %s", entry.Clock.ID, decodedEntry.Entry.Clock.ID)
	}
	if decodedEntry.Entry.Clock.Time != entry.Clock.Time {
		t.Errorf("Expected Clock Time %d, got %d", entry.Clock.Time, decodedEntry.Entry.Clock.Time)
	}
	if decodedEntry.Entry.V != entry.V {
		t.Errorf("Expected Version %d, got %d", entry.V, decodedEntry.Entry.V)
	}
	if decodedEntry.Entry.Key != entry.Key {
		t.Errorf("Expected Key %s, got %s", entry.Key, decodedEntry.Entry.Key)
	}
	if decodedEntry.Entry.Identity != entry.Identity {
		t.Errorf("Expected Identity %s, got %s", entry.Identity, decodedEntry.Entry.Identity)
	}
	if decodedEntry.Entry.Signature != entry.Signature {
		t.Errorf("Expected Signature %s, got %s", entry.Signature, decodedEntry.Entry.Signature)
	}

	// Check that the CBOR bytes match between encoding and decoding
	if !bytes.Equal(encodedEntry.Bytes, decodedEntry.Bytes) {
		t.Errorf("Encoded bytes do not match decoded bytes")
	}
}
