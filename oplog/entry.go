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
	"orbitdb/go-orbitdb/identities/identitytypes"
	"sort"
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

func NewEntry(identity *identitytypes.Identity, id string, payload string, clock Clock, next []string, refs []string) EncodedEntry {
	if identity == nil {
		panic("Identity is required, cannot create entry")
	}
	if id == "" || payload == "" {
		panic("Entry requires an id and payload")
	}
	// Initialize next and refs as empty slices if nil
	if next == nil {
		next = []string{}
	} else {
		sort.Strings(next)
	}
	if refs == nil {
		refs = []string{}
	} else {
		sort.Strings(refs)
	}

	// Create an entry without Key, Identity, and Signature
	entry := Entry{
		ID:      id,
		Payload: payload,
		Next:    next,
		Refs:    refs,
		Clock:   clockOrDefault(clock, identity),
		V:       2,
	}

	// Encode the entry to CBOR
	encodedEntry := Encode(entry)

	// Sign the encoded entry data
	signature, err := identity.Sign(encodedEntry.Bytes.Bytes())
	if err != nil {
		panic(err)
	}

	// Now assign Key, Identity, and Signature fields
	entry.Key = identity.PublicKey
	entry.Identity = identity.Hash
	entry.Signature = signature

	// Re-encode the entry with the newly assigned fields
	finalEncodedEntry := Encode(entry)
	return finalEncodedEntry
}

func VerifyEntrySignature(identity *identitytypes.Identity, entry EncodedEntry) bool {

	// Recreate the entry data without Signature, Key, and Identity fields
	entryData := Entry{
		ID:      entry.ID,
		Payload: entry.Payload,
		Next:    entry.Next,
		Refs:    entry.Refs,
		Clock:   entry.Clock,
		V:       entry.V,
	}

	// Encode the entry data without the Key, Identity, and Signature fields
	reconstructedEncodedEntry := Encode(entryData)

	// Use the provider to verify the signature on the reconstructed data
	return identity.Verify(entry.Signature, reconstructedEncodedEntry.Bytes.Bytes())
}

// IsEntry checks if an object is a valid entry
func IsEntry(entry Entry) bool {
	return entry.ID != "" && entry.Payload != "" && entry.Clock.ID != "" && entry.Clock.Time > 0
}

// IsEqual checks if two entries are equal based on their hash values
func IsEqual(a EncodedEntry, b EncodedEntry) bool {
	return bytes.Equal(a.Bytes.Bytes(), b.Bytes.Bytes())
}

// Encode encodes the entry into CBOR and returns an EncodedEntry
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

// Helper function to set a default clock if not provided
func clockOrDefault(clock Clock, identity *identitytypes.Identity) Clock {
	if clock.ID == "" {
		return Clock{ID: identity.PublicKey, Time: 1}
	}
	return clock
}
