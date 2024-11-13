package main

import (
	"encoding/hex"
	"fmt"
	"log"
	"orbitdb/go-orbitdb/identities/identitytypes"
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

	// Define the test identity details
	id := "0x01234567890abcdefghijklmnopqrstuvwxyz"
	publicKey := "<pubkey>"
	signatures := map[string]string{
		"id":        "signature for <id>",
		"publicKey": "signature for <publicKey + idSignature>",
	}
	idType := "orbitdb"

	// Create an Identity struct without a PrivateKey
	testIdentity := identitytypes.Identity{
		ID:         id,
		PublicKey:  publicKey,
		Signatures: signatures,
		Type:       idType,
	}

	// Encode the identity to get the CBOR-encoded bytes and CID hash
	hash, encodedBytes, err := identitytypes.EncodeIdentity(testIdentity)
	if err != nil {
		log.Fatalf("Error encoding identity: %v", err)
	}

	// Display the encoded values
	fmt.Printf("Encoded Hash: %s\n", hash)
	fmt.Printf("Encoded Bytes (CBOR): %x\n", encodedBytes)
	// expected hash: 'zdpuArx43BnXdDff5rjrGLYrxUomxNroc2uaocTgcWK76UfQT'
}
