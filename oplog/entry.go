package oplog

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/hex"
	"github.com/ipfs/go-cid"
	"github.com/ipld/go-ipld-prime/codec/dagcbor"
	"github.com/ipld/go-ipld-prime/node/basicnode"
	"github.com/multiformats/go-multibase"
	mh "github.com/multiformats/go-multihash"
	"math/big"
	"orbitdb/go-orbitdb/identities/identitytypes"
	"orbitdb/go-orbitdb/keystore"
	"sort"
)

type Entry struct {
	ID        string   `json:"id"`
	Payload   string   `json:"payload"`
	Next      []string `json:"next"`
	Refs      []string `json:"refs"`
	Clock     Clock    `json:"clock"`
	V         int      `json:"v"`
	Key       string   `json:"key"`
	Identity  string   `json:"identity"`
	Signature string   `json:"sig"`
}

type EncodedEntry struct {
	Entry
	Bytes []byte
	CID   cid.Cid
	Hash  string
}

func (e EncodedEntry) GetBase58CID() string {
	// Convert CID to base58btc encoding
	cidBase58, _ := e.CID.StringOfBase(multibase.Base58BTC)
	return cidBase58
}

// NewEntry creates a new log entry, signing it with the KeyStore.
func NewEntry(ks *keystore.KeyStore, identity *identitytypes.Identity, id string, payload string, clock Clock, next []string, refs []string) EncodedEntry {
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
	signature, err := ks.SignMessage(identity.ID, encodedEntry.Bytes)
	if err != nil {
		panic(err)
	}

	// Now assign Key, Identity, and Signature fields
	entry.Key = identity.PublicKey
	entry.Identity = identity.Hash
	entry.Signature = signature

	return EncodedEntry{
		Entry: entry,
		Bytes: encodedEntry.Bytes,
		CID:   encodedEntry.CID,
		Hash:  encodedEntry.Hash,
	}
}

// VerifyEntrySignature verifies the signature on an entry using KeyStore.
func VerifyEntrySignature(ks *keystore.KeyStore, identity *identitytypes.Identity, entry EncodedEntry) bool {
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

	// Decode the hex-encoded public key to reconstruct the ecdsa.PublicKey
	publicKeyBytes, err := hex.DecodeString(identity.PublicKey)
	if err != nil || len(publicKeyBytes) < 64 {
		return false
	}

	// Reconstruct the ecdsa.PublicKey
	pubKey := ecdsa.PublicKey{
		Curve: elliptic.P256(),
		X:     new(big.Int).SetBytes(publicKeyBytes[:len(publicKeyBytes)/2]),
		Y:     new(big.Int).SetBytes(publicKeyBytes[len(publicKeyBytes)/2:]),
	}

	// Use the KeyStore instance to verify the signature
	verified, err := ks.VerifyMessage(pubKey, reconstructedEncodedEntry.Bytes, entry.Signature)
	return err == nil && verified
}

// IsEntry checks if an object is a valid entry
func IsEntry(entry Entry) bool {
	return entry.ID != "" && entry.Payload != "" && entry.Clock.ID != "" && entry.Clock.Time > 0
}

// IsEqual checks if two entries are equal based on their hash values
func IsEqual(a EncodedEntry, b EncodedEntry) bool {
	return a.Hash == b.Hash
}

// Encode encodes the entry into CBOR and returns an EncodedEntry
func Encode(entry Entry) EncodedEntry {
	// Create a basic map node for encoding
	nb := basicnode.Prototype__Map{}.NewBuilder()
	ma, _ := nb.BeginMap(9)

	// Assemble each field
	ma.AssembleKey().AssignString("id")
	ma.AssembleValue().AssignString(entry.ID)

	ma.AssembleKey().AssignString("payload")
	ma.AssembleValue().AssignString(entry.Payload)

	ma.AssembleKey().AssignString("next")
	na, _ := ma.AssembleValue().BeginList(int64(len(entry.Next)))
	for _, n := range entry.Next {
		na.AssembleValue().AssignString(n)
	}
	na.Finish()

	ma.AssembleKey().AssignString("refs")
	ra, _ := ma.AssembleValue().BeginList(int64(len(entry.Refs)))
	for _, r := range entry.Refs {
		ra.AssembleValue().AssignString(r)
	}
	ra.Finish()

	ma.AssembleKey().AssignString("clock")
	ca, _ := ma.AssembleValue().BeginMap(2)
	ca.AssembleKey().AssignString("id")
	ca.AssembleValue().AssignString(entry.Clock.ID)
	ca.AssembleKey().AssignString("time")
	ca.AssembleValue().AssignInt(int64(entry.Clock.Time))
	ca.Finish()

	ma.AssembleKey().AssignString("v")
	ma.AssembleValue().AssignInt(int64(entry.V))

	ma.AssembleKey().AssignString("key")
	ma.AssembleValue().AssignString(entry.Key)

	ma.AssembleKey().AssignString("identity")
	ma.AssembleValue().AssignString(entry.Identity)

	ma.AssembleKey().AssignString("sig")
	ma.AssembleValue().AssignString(entry.Signature)

	ma.Finish()

	// Get the final built node
	node := nb.Build()

	// Encode to CBOR
	var buf bytes.Buffer
	if err := dagcbor.Encode(node, &buf); err != nil {
		panic(err)
	}

	// Calculate CID for CBOR-encoded bytes
	hash, err := mh.Sum(buf.Bytes(), mh.SHA2_256, -1)
	if err != nil {
		panic(err)
	}
	c := cid.NewCidV1(cid.DagCBOR, hash)

	// Encode CID to base58btc for the hash
	hashStr, err := c.StringOfBase(multibase.Base58BTC)
	if err != nil {
		panic(err)
	}

	return EncodedEntry{Entry: entry, Bytes: buf.Bytes(), CID: c, Hash: hashStr}
}

// Decode decodes CBOR-encoded data into an EncodedEntry struct
func Decode(encodedData []byte) (EncodedEntry, error) {
	// Create a node prototype for decoding
	nb := basicnode.Prototype__Map{}.NewBuilder()
	buf := bytes.NewReader(encodedData)

	// Decode the CBOR data
	if err := dagcbor.Decode(nb, buf); err != nil {
		return EncodedEntry{}, err
	}
	node := nb.Build()

	// Extract values from the node
	entry := Entry{}
	if idNode, err := node.LookupByString("id"); err == nil {
		id, _ := idNode.AsString()
		entry.ID = id
	}
	if payloadNode, err := node.LookupByString("payload"); err == nil {
		payload, _ := payloadNode.AsString()
		entry.Payload = payload
	}
	if vNode, err := node.LookupByString("v"); err == nil {
		v, _ := vNode.AsInt()
		entry.V = int(v) // Cast int64 to int
	}
	if keyNode, err := node.LookupByString("key"); err == nil {
		key, _ := keyNode.AsString()
		entry.Key = key
	}
	if identityNode, err := node.LookupByString("identity"); err == nil {
		identity, _ := identityNode.AsString()
		entry.Identity = identity
	}
	if sigNode, err := node.LookupByString("sig"); err == nil {
		sig, _ := sigNode.AsString()
		entry.Signature = sig
	}

	// Decode nested Clock
	if clockNode, err := node.LookupByString("clock"); err == nil {
		clock := Clock{}
		if clockIDNode, err := clockNode.LookupByString("id"); err == nil {
			clockID, _ := clockIDNode.AsString()
			clock.ID = clockID
		}
		if clockTimeNode, err := clockNode.LookupByString("time"); err == nil {
			clockTime, _ := clockTimeNode.AsInt()
			clock.Time = int(clockTime) // Cast int64 to int
		}
		entry.Clock = clock
	}

	// Decode lists (Next and Refs)
	if nextNode, err := node.LookupByString("next"); err == nil {
		nextLen := nextNode.Length()
		for i := int64(0); i < nextLen; i++ {
			nNode, _ := nextNode.LookupByIndex(i)
			n, _ := nNode.AsString()
			entry.Next = append(entry.Next, n)
		}
	}
	if refsNode, err := node.LookupByString("refs"); err == nil {
		refsLen := refsNode.Length()
		for i := int64(0); i < refsLen; i++ {
			rNode, _ := refsNode.LookupByIndex(i)
			r, _ := rNode.AsString()
			entry.Refs = append(entry.Refs, r)
		}
	}

	// Calculate the CID for CBOR-encoded bytes
	hash, err := mh.Sum(encodedData, mh.SHA2_256, -1)
	if err != nil {
		return EncodedEntry{}, err
	}
	c := cid.NewCidV1(cid.DagCBOR, hash)
	hashStr, _ := c.StringOfBase(multibase.Base58BTC)

	return EncodedEntry{
		Entry: entry,
		Bytes: encodedData,
		CID:   c,
		Hash:  hashStr,
	}, nil
}

// Helper function to set a default clock if not provided
func clockOrDefault(clock Clock, identity *identitytypes.Identity) Clock {
	if clock.ID == "" {
		return Clock{ID: identity.PublicKey, Time: 1}
	}
	return clock
}
