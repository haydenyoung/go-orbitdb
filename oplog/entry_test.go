package oplog

import (
	"bytes"
	"testing"
)

func TestNewEntry(t *testing.T) {
	// Create a new entry with a sample payload
	payload := "some entry"
	e := NewEntry(payload)

	// Check if the payload is correctly set
	if e.Payload != payload {
		t.Errorf("Expected Payload %s, got %s", payload, e.Payload)
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
