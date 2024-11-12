package main

import (
	"encoding/hex"
	"fmt"
	"orbitdb/go-orbitdb/oplog"
)

func main() {
	// Define the same entry data as in JavaScript
	entry := oplog.Entry{
		ID:        "test-log-id",
		Payload:   "hello world",
		Next:      []string{},
		Refs:      []string{},
		Clock:     oplog.Clock{ID: "test-user-id", Time: 0},
		V:         2,
		Key:       "test-public-key",
		Identity:  "test-identity-hash",
		Signature: "test-signature",
	}

	// Encode the entry using your Encode function
	encodedEntry := oplog.Encode(entry)
	fmt.Println("CID in Go:", encodedEntry.CID)
	// Output the bytes in hex format for easy comparison
	fmt.Println("CBOR Encoded Bytes in Go:", hex.EncodeToString(encodedEntry.Bytes))
}
