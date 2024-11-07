package oplog

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"github.com/ipfs/go-cid"
	"github.com/ipld/go-ipld-prime"
	"github.com/ipld/go-ipld-prime/codec/dagcbor"
	"github.com/ipld/go-ipld-prime/node/bindnode"
	"github.com/multiformats/go-multibase"
	mh "github.com/multiformats/go-multihash"
	"orbitdb/go-orbitdb/identities"
	"orbitdb/go-orbitdb/identities/provider_registry"
)

type Entry struct {
	ID        string
	Payload   string
	Next      []string
	Refs      []string
	Clock     Clock
	V         int
	Key       string // Public key of the identity
	Identity  string // Identity hash or identifier
	Signature string // Signature of the entry
}

type EncodedEntry struct {
	Entry
	Bytes bytes.Buffer
	CID   cid.Cid
}

func (e EncodedEntry) GetBase58CID() string {
	// Convert CID to base58btc encoding
	cidBase58, _ := e.CID.StringOfBase(multibase.Base58BTC)
	return cidBase58
}

func NewEntry(identity *identities.Identity, id string, payload string, clock Clock) EncodedEntry {
	if identity == nil {
		panic("Identity is required, cannot create entry")
	}
	if id == "" {
		panic("Entry requires an id")
	}
	if payload == "" {
		panic("Entry requires a payload")
	}

	entry := Entry{
		ID:       id,
		Payload:  payload,
		Clock:    clock,
		V:        2,
		Key:      identity.PublicKeyHex(), // Convert public key to hex string for storage
		Identity: identity.Identity,       // Use the identity's identifier (hash)
		Next:     []string{},              // Initialize Next as empty array
		Refs:     []string{},              // Initialize Refs as empty array
	}

	// Encode the entry to CBOR
	encodedEntry := Encode(entry)

	// Sign the encoded entry data
	signature, err := identity.Sign(encodedEntry.Bytes.Bytes())
	if err != nil {
		panic(err)
	}

	// Set the signature in the encoded entry
	encodedEntry.Entry.Signature = signature

	return encodedEntry
}

func VerifyEntrySignature(identity *identities.Identity, entry EncodedEntry) bool {
	// Retrieve the identity provider for the identity type
	provider, err := provider_registry.GetIdentityProvider(identity.Type)
	if err != nil {
		return false // Provider not found or error retrieving it
	}

	// Use the provider to verify the identity by checking the entry's data and signature
	valid, err := provider.VerifyIdentityWithEntry(identity, entry.Bytes.Bytes(), entry.Signature)
	if err != nil {
		return false
	}
	return valid
}

func Encode(entry Entry) EncodedEntry {
	// Define the schema for Entry, including the new fields
	ts, err := ipld.LoadSchemaBytes([]byte(`
		type Clock struct {
			id String
			time Int
		} representation map

		type Entry struct {
			id String
			payload String
			next [String]
			refs [String]
			clock Clock
			v Int
			key String
			identity String
			sig String
		} representation map
	`))
	if err != nil {
		panic(err)
	}

	schemaType := ts.TypeByName("Entry")
	node := bindnode.Wrap(&entry, schemaType)

	var buf bytes.Buffer
	if err := dagcbor.Encode(node.Representation(), &buf); err != nil {
		panic(err)
	}

	fmt.Println("Raw CBOR Encoded Bytes in Go (Hex):", hex.EncodeToString(buf.Bytes()))

	// Hash the bytes and generate a CID
	hash, err := mh.Sum(buf.Bytes(), mh.SHA2_256, -1) // SHA-256 hash
	if err != nil {
		panic(err)
	}

	c := cid.NewCidV1(cid.DagCBOR, hash) // Create CID with DAG-CBOR codec

	// Return the EncodedEntry with CID
	return EncodedEntry{Entry: entry, Bytes: buf, CID: c}
}
