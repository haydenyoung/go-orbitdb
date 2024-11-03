package oplog

import (
    "bytes"
    "github.com/ipld/go-ipld-prime"
    "github.com/ipld/go-ipld-prime/node/bindnode"
    "github.com/ipld/go-ipld-prime/codec/dagcbor"
)

type Entry struct {
    Payload string
}

type EncodedEntry struct {
    Entry
    Bytes bytes.Buffer
}

func NewEntry(payload string) EncodedEntry {
    entry := Entry{Payload: payload}
    return Encode(entry)
}

func Encode(entry Entry) EncodedEntry {
    ts, err := ipld.LoadSchemaBytes([]byte(`
        type Entry struct {
            payload String
        } representation tuple
    `))
    if err != nil {
        panic(err)
    }

    
    schemaType := ts.TypeByName("Entry")
    node := bindnode.Wrap(&entry, schemaType)   

    var buf bytes.Buffer
    dagcbor.Encode(node.Representation(), &buf)
    
    encodedEntry := EncodedEntry{Entry: entry, Bytes: buf}
        
    return encodedEntry
}