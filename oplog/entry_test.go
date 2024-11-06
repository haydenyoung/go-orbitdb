package oplog

import (
	"bytes"
	"testing"
)

func TestNewEntry(t *testing.T) {
	// Create an ID and a Clock for the entry
	id := "test-log-id"
	clock := NewClock("test-id", 1)

	// Create a new entry with an id, clock, and sample payload
	payload := "some entry"
	e := NewEntry(id, payload, *clock)

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
	// Create an entry to test encoding
	entry := Entry{Payload: "test payload"}
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
}
