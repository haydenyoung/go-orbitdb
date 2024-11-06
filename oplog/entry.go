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
	ID      string
	Payload string
	Clock   Clock
	V       int
}

type EncodedEntry struct {
	Entry
	Bytes bytes.Buffer
	CID   cid.Cid
}

func NewEntry(id string, payload string, clock Clock) EncodedEntry {
	entry := Entry{
		ID:      id,
		Payload: payload,
		Clock:   clock,
		V:       2,
	}
	return Encode(entry)
}

func Encode(entry Entry) EncodedEntry {
	// Define the schema and encode the entry to CBOR format
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
