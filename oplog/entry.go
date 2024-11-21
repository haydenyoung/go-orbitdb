package oplog

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/hex"
	"github.com/ipfs/go-cid"
	"github.com/ipld/go-ipld-prime/codec/dagcbor"
	"github.com/ipld/go-ipld-prime/datamodel"
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

	return Encode(entry)
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

// IsEqual checks if two entries are equal. Exclude Signature, Hash, and Bytes from the comparison since they can differ even if the entries have the same content.
// The reason is The ECDSA algorithm uses a random value (k) during the signing process to ensure that each signature is unique and secure.
// Even if the same message is signed multiple times with the same private key, the signatures will be different due to this randomness.
func IsEqual(entry1 EncodedEntry, entry2 EncodedEntry) bool {
	return entry1.Entry.ID == entry2.Entry.ID &&
		entry1.Entry.Payload == entry2.Entry.Payload &&
		EqualStringSlices(entry1.Entry.Next, entry2.Entry.Next) &&
		EqualStringSlices(entry1.Entry.Refs, entry2.Entry.Refs) &&
		entry1.Entry.Clock.ID == entry2.Entry.Clock.ID &&
		entry1.Entry.Clock.Time == entry2.Entry.Clock.Time &&
		entry1.Entry.V == entry2.Entry.V &&
		entry1.Entry.Key == entry2.Entry.Key &&
		entry1.Entry.Identity == entry2.Entry.Identity
}

func EqualStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// Encode encodes the entry into CBOR and returns an EncodedEntry
func Encode(entry Entry) EncodedEntry {
	// Create a basic map node for encoding
	nb := basicnode.Prototype__Map{}.NewBuilder()
	ma, err := nb.BeginMap(9)
	if err != nil {
		panic(err)
	}

	// Assemble each field using helper functions
	if err := assembleStringField(ma, "id", entry.ID); err != nil {
		panic(err)
	}

	if err := assembleStringField(ma, "payload", entry.Payload); err != nil {
		panic(err)
	}

	if err := assembleStringList(ma, "next", entry.Next); err != nil {
		panic(err)
	}

	if err := assembleStringList(ma, "refs", entry.Refs); err != nil {
		panic(err)
	}

	if err := assembleClock(ma, "clock", entry.Clock); err != nil {
		panic(err)
	}

	if err := assembleIntField(ma, "v", int64(entry.V)); err != nil {
		panic(err)
	}

	if err := assembleStringField(ma, "key", entry.Key); err != nil {
		panic(err)
	}

	if err := assembleStringField(ma, "identity", entry.Identity); err != nil {
		panic(err)
	}

	if err := assembleStringField(ma, "sig", entry.Signature); err != nil {
		panic(err)
	}

	// Finish assembling the map
	if err := ma.Finish(); err != nil {
		panic(err)
	}

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
	// Create a node builder for decoding
	nb := basicnode.Prototype.Any.NewBuilder()
	buf := bytes.NewReader(encodedData)

	// Decode the CBOR data
	if err := dagcbor.Decode(nb, buf); err != nil {
		return EncodedEntry{}, err
	}
	node := nb.Build()

	// Extract values from the node using helper functions
	var entry Entry
	var err error

	if entry.ID, err = getString(node, "id"); err != nil {
		return EncodedEntry{}, err
	}
	if entry.Payload, err = getString(node, "payload"); err != nil {
		return EncodedEntry{}, err
	}
	if v, err := getInt(node, "v"); err == nil {
		entry.V = int(v)
	} else {
		return EncodedEntry{}, err
	}
	if entry.Key, err = getString(node, "key"); err != nil {
		return EncodedEntry{}, err
	}
	if entry.Identity, err = getString(node, "identity"); err != nil {
		return EncodedEntry{}, err
	}
	if entry.Signature, err = getString(node, "sig"); err != nil {
		return EncodedEntry{}, err
	}

	// Decode nested Clock
	clockNode, err := node.LookupByString("clock")
	if err != nil {
		return EncodedEntry{}, err
	}
	if entry.Clock.ID, err = getString(clockNode, "id"); err != nil {
		return EncodedEntry{}, err
	}
	if timeVal, err := getInt(clockNode, "time"); err == nil {
		entry.Clock.Time = int(timeVal)
	} else {
		return EncodedEntry{}, err
	}

	// Decode lists (Next and Refs)
	if entry.Next, err = getStringList(node, "next"); err != nil {
		return EncodedEntry{}, err
	}
	if entry.Refs, err = getStringList(node, "refs"); err != nil {
		return EncodedEntry{}, err
	}

	// Calculate the CID for CBOR-encoded bytes
	hash, err := mh.Sum(encodedData, mh.SHA2_256, -1)
	if err != nil {
		return EncodedEntry{}, err
	}
	c := cid.NewCidV1(cid.DagCBOR, hash)
	hashStr, err := c.StringOfBase(multibase.Base58BTC)
	if err != nil {
		return EncodedEntry{}, err
	}

	return EncodedEntry{
		Entry: entry,
		Bytes: encodedData,
		CID:   c,
		Hash:  hashStr,
	}, nil
}

func assembleStringField(ma datamodel.MapAssembler, key string, value string) error {
	if err := ma.AssembleKey().AssignString(key); err != nil {
		return err
	}
	if err := ma.AssembleValue().AssignString(value); err != nil {
		return err
	}
	return nil
}

func assembleIntField(ma datamodel.MapAssembler, key string, value int64) error {
	if err := ma.AssembleKey().AssignString(key); err != nil {
		return err
	}
	if err := ma.AssembleValue().AssignInt(value); err != nil {
		return err
	}
	return nil
}

func assembleStringList(ma datamodel.MapAssembler, key string, values []string) error {
	if err := ma.AssembleKey().AssignString(key); err != nil {
		return err
	}
	la, err := ma.AssembleValue().BeginList(int64(len(values)))
	if err != nil {
		return err
	}
	for _, v := range values {
		if err := la.AssembleValue().AssignString(v); err != nil {
			return err
		}
	}
	if err := la.Finish(); err != nil {
		return err
	}
	return nil
}

func assembleClock(ma datamodel.MapAssembler, key string, clock Clock) error {
	if err := ma.AssembleKey().AssignString(key); err != nil {
		return err
	}
	ca, err := ma.AssembleValue().BeginMap(2)
	if err != nil {
		return err
	}
	if err := assembleStringField(ca, "id", clock.ID); err != nil {
		return err
	}
	if err := assembleIntField(ca, "time", int64(clock.Time)); err != nil {
		return err
	}
	if err := ca.Finish(); err != nil {
		return err
	}
	return nil
}

func getString(node datamodel.Node, key string) (string, error) {
	childNode, err := node.LookupByString(key)
	if err != nil {
		return "", err
	}
	return childNode.AsString()
}

func getInt(node datamodel.Node, key string) (int64, error) {
	childNode, err := node.LookupByString(key)
	if err != nil {
		return 0, err
	}
	return childNode.AsInt()
}

func getStringList(node datamodel.Node, key string) ([]string, error) {
	listNode, err := node.LookupByString(key)
	if err != nil {
		return nil, err
	}
	length := listNode.Length()

	var list []string
	for i := int64(0); i < length; i++ {
		itemNode, err := listNode.LookupByIndex(i)
		if err != nil {
			return nil, err
		}
		str, err := itemNode.AsString()
		if err != nil {
			return nil, err
		}
		list = append(list, str)
	}
	return list, nil
}

// Helper function to set a default clock if not provided
func clockOrDefault(clock Clock, identity *identitytypes.Identity) Clock {
	if clock.ID == "" {
		return Clock{ID: identity.PublicKey, Time: 1}
	}
	return clock
}
