package oplog

import (
	"bytes"
	"orbitdb/go-orbitdb/identities"
	"orbitdb/go-orbitdb/identities/provider_registry"
	"orbitdb/go-orbitdb/identities/providers"
	"testing"
)

func TestNewEntry(t *testing.T) {
	// Register the PublicKeyIdentityProvider if not already registered
	err := provider_registry.UseIdentityProvider(providers.NewPublicKeyIdentityProvider())
	if err != nil && err.Error() != "identity provider already registered" {
		t.Fatalf("Failed to register identity provider: %v", err)
	}

	// Create an identity for the entry
	identity, err := identities.NewIdentity()
	if err != nil {
		t.Fatalf("Failed to create identity: %v", err)
	}

	// Create an ID and a Clock for the entry
	id := "test-log-id"
	clock := NewClock("test-id", 1)

	// Create a new entry with an identity, id, clock, and sample payload
	payload := "some entry"
	e := NewEntry(identity, id, payload, *clock)

	// Check if the ID is correctly set
	if e.ID != id {
		t.Errorf("Expected ID %s, got %s", id, e.ID)
	}

	// Check if the payload is correctly set
	if e.Payload != payload {
		t.Errorf("Expected Payload %s, got %s", payload, e.Payload)
	}

	// Check if the clock is correctly set
	if e.Clock.id != clock.id || e.Clock.time != clock.time {
		t.Errorf("Expected Clock %v, got %v", clock, e.Clock)
	}

	// Check if the version is set to 2
	if e.V != 2 {
		t.Errorf("Expected version 2, got %d", e.V)
	}

	// Check if the key (public key) is correctly set
	expectedKey := identity.PublicKeyHex()
	if e.Key != expectedKey {
		t.Errorf("Expected Key %s, got %s", expectedKey, e.Key)
	}

	// Check if the identity field matches the identity's identifier
	if e.Identity != identity.Identity {
		t.Errorf("Expected Identity %s, got %s", identity.Identity, e.Identity)
	}

	// Verify that the entry signature is non-empty
	if e.Signature == "" {
		t.Error("Expected non-empty signature")
	}

	// Verify the signature is valid
	if !VerifyEntrySignature(identity, e) {
		t.Error("Signature verification failed")
	}

	// Check if the CBOR bytes are non-empty
	if e.Bytes.Len() == 0 {
		t.Error("Expected non-empty Bytes buffer after encoding")
	}

	// Verify the CID is valid and non-empty
	if e.CID.String() == "" {
		t.Error("Expected a valid CID, but got an empty string")
	}
}

func TestEncode(t *testing.T) {
	// Create an identity for the entry
	identity, err := identities.NewIdentity()
	if err != nil {
		t.Fatalf("Failed to create identity: %v", err)
	}

	// Create an entry to test encoding with id, payload, clock, key, and identity
	id := "test-log-id"
	payload := "test payload"
	clock := NewClock("test-id", 1)
	entry := Entry{
		ID:       id,
		Payload:  payload,
		Clock:    *clock,
		V:        2,
		Key:      identity.PublicKeyHex(),
		Identity: identity.Identity,
	}

	// Encode the entry
	encodedEntry := Encode(entry)

	// Verify the CBOR-encoded bytes
	var expectedBytes bytes.Buffer
	expectedBytes.Write(encodedEntry.Bytes.Bytes())

	if !bytes.Equal(encodedEntry.Bytes.Bytes(), expectedBytes.Bytes()) {
		t.Error("Encoded bytes do not match expected output")
	}

	// Verify CID generation
	if encodedEntry.CID.String() == "" {
		t.Error("Expected a valid CID, but got an empty string")
	}

	// Check if the ID is correctly encoded
	if encodedEntry.ID != id {
		t.Errorf("Expected ID %s, got %s", id, encodedEntry.ID)
	}

	// Check if the version is correctly encoded as 2
	if encodedEntry.V != 2 {
		t.Errorf("Expected version 2, got %d", encodedEntry.V)
	}

	// Check if the key (public key) is correctly encoded
	if encodedEntry.Key != identity.PublicKeyHex() {
		t.Errorf("Expected Key %s, got %s", identity.PublicKeyHex(), encodedEntry.Key)
	}

	// Check if the identity field matches the identity's identifier
	if encodedEntry.Identity != identity.Identity {
		t.Errorf("Expected Identity %s, got %s", identity.Identity, encodedEntry.Identity)
	}
}
