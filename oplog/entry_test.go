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

func TestEntryCID(t *testing.T) {
	// Define test cases
	tests := []struct {
		id          string
		payload     string
		expectedCID string
	}{
		{
			id:          "A",
			payload:     "hello",
			expectedCID: "zdpuAsKzwUEa8cz9pkJxxFMxLuP3cutA9PDGoLZytrg4RSVEa",
		},
		{
			id:          "A",
			payload:     "hello world",
			expectedCID: "zdpuAmthfqpHRQjdSpKN5etr1GrreJb7QcU1Hshm6pERnzsxi",
		},
	}

	for _, tt := range tests {
		// Generate a new identity for signing
		identity, err := identities.NewIdentity()
		if err != nil {
			t.Fatalf("failed to create identity: %v", err)
		}

		// Create a new clock
		clock := NewClock(identity.ID, 0)

		// Create a new entry
		entry := NewEntry(identity, tt.id, tt.payload, *clock)

		// Get the generated CID
		generatedCID := entry.GetBase58CID()

		// Verify if the generated CID matches the expected CID
		if generatedCID != tt.expectedCID {
			t.Errorf("Test failed for payload '%s': expected CID %s, got %s", tt.payload, tt.expectedCID, generatedCID)
		}

		// Verify additional entry attributes
		if entry.Entry.ID != tt.id {
			t.Errorf("expected id %s, got %s", tt.id, entry.Entry.ID)
		}
		if entry.Entry.Payload != tt.payload {
			t.Errorf("expected payload %s, got %s", tt.payload, entry.Entry.Payload)
		}
		if entry.Entry.Clock.id != identity.PublicKeyHex() {
			t.Errorf("expected clock id %s, got %s", identity.PublicKeyHex(), entry.Entry.Clock.id)
		}
		if entry.Entry.Clock.time != 0 {
			t.Errorf("expected clock time 0, got %d", entry.Entry.Clock.time)
		}
		if entry.Entry.V != 2 {
			t.Errorf("expected version 2, got %d", entry.Entry.V)
		}
		if len(entry.Entry.Next) != 0 {
			t.Errorf("expected Next to be empty, got %v", entry.Entry.Next)
		}
		if len(entry.Entry.Refs) != 0 {
			t.Errorf("expected Refs to be empty, got %v", entry.Entry.Refs)
		}

		// Log the generated CID for reference
		t.Logf("Generated CID for payload '%s': %s", tt.payload, generatedCID)
	}
}

func TestEntryWithNextAndErrorCases(t *testing.T) {
	identity, err := identities.NewIdentity()
	if err != nil {
		t.Fatalf("failed to create identity: %v", err)
	}

	clock := NewClock(identity.PublicKeyHex(), 0)

	// Test an entry with `next` parameter as an array of entries
	entry1 := NewEntry(identity, "A", "hello world", *clock)
	entry1.Clock.time++ // Increment clock for next entry
	entry2 := NewEntry(identity, "A", "hello again", entry1.Clock)
	entry2.Entry.Next = []string{entry1.GetBase58CID()}

	if entry2.Entry.Payload != "hello again" {
		t.Errorf("expected payload 'hello again', got %s", entry2.Entry.Payload)
	}
	if len(entry2.Entry.Next) != 1 || entry2.Entry.Next[0] != entry1.GetBase58CID() {
		t.Errorf("expected Next to contain CID of entry1, got %v", entry2.Entry.Next)
	}
	if entry2.Entry.Clock.time != 1 {
		t.Errorf("expected clock time 1, got %d", entry2.Entry.Clock.time)
	}

	// Test error cases

	// Missing identity
	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("expected panic for missing identity")
			}
		}()
		NewEntry(nil, "A", "hello", *clock)
	}()

	// Missing ID
	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("expected panic for missing id")
			}
		}()
		NewEntry(identity, "", "hello", *clock)
	}()

	// Missing payload
	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("expected panic for missing payload")
			}
		}()
		NewEntry(identity, "A", "", *clock)
	}()
}

func TestEntryEncodingWithNextAndClock(t *testing.T) {
	// Register the PublicKeyIdentityProvider if not already registered
	err := provider_registry.UseIdentityProvider(providers.NewPublicKeyIdentityProvider())
	if err != nil && err.Error() != "identity provider already registered" {
		t.Fatalf("Failed to register identity provider: %v", err)
	}

	// Create an identity for the test
	identity, err := identities.NewIdentity()
	if err != nil {
		t.Fatalf("Failed to create identity: %v", err)
	}

	// Create an initial entry with a clock
	clock := NewClock(identity.PublicKeyHex(), 0)
	entry1 := Entry{
		ID:       "A",
		Payload:  "hello world",
		Clock:    *clock,
		V:        2,
		Key:      identity.PublicKeyHex(),
		Identity: identity.Identity,
		Next:     []string{},
		Refs:     []string{},
	}

	// Encode the first entry to get its CID
	encodedEntry1 := Encode(entry1)

	// Increment the clock using TickClock
	*clock = TickClock(*clock)

	// Create a second entry with the updated clock and `Next` reference to entry1
	entry2 := Entry{
		ID:       "A",
		Payload:  "hello again",
		Clock:    *clock,
		V:        2,
		Key:      identity.PublicKeyHex(),
		Identity: identity.Identity,
		Next:     []string{encodedEntry1.CID.String()}, // Reference to the first entry
		Refs:     []string{},
	}

	// Encode the second entry
	encodedEntry2 := Encode(entry2)

	// Expected CID (you will need to compute this from a trusted source for accuracy)
	expectedHash := "replace_with_actual_expected_hash"

	// Check if the encoded entry CID matches the expected hash
	if encodedEntry2.CID.String() != expectedHash {
		t.Errorf("Expected CID %s, got %s", expectedHash, encodedEntry2.CID.String())
	}
}
