package oplog

import (
	"bytes"
	"github.com/ipfs/go-cid"
	"github.com/ipld/go-ipld-prime"
	"github.com/ipld/go-ipld-prime/codec/dagcbor"
	"github.com/ipld/go-ipld-prime/node/bindnode"
	mh "github.com/multiformats/go-multihash"
)

type Entry struct {
	ID        string
	Payload   string
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

func NewEntry(identity *Identity, id string, payload string, clock Clock) EncodedEntry {
	entry := Entry{
		ID:       id,
		Payload:  payload,
		Clock:    clock,
		V:        2,
		Key:      identity.PublicKeyHex(), // Convert public key to hex string for storage
		Identity: identity.Identity,       // Use the identity's identifier (hash)
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

func VerifyEntrySignature(identity *Identity, entry EncodedEntry) bool {
	// Verify the signature using the provided identity
	valid, err := identity.VerifySignature(entry.Bytes.Bytes(), entry.Signature)
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
			clock Clock
			v Int
			key String
			identity String
			signature String
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

	// Hash the bytes and generate a CID
	hash, err := mh.Sum(buf.Bytes(), mh.SHA2_256, -1) // SHA-256 hash
	if err != nil {
		panic(err)
	}

	c := cid.NewCidV1(cid.DagCBOR, hash) // Create CID with DAG-CBOR codec

	// Return the EncodedEntry with CID
	return EncodedEntry{Entry: entry, Bytes: buf, CID: c}
}
