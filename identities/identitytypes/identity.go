package identitytypes

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/ipfs/go-cid"
	"github.com/ipld/go-ipld-prime"
	"github.com/ipld/go-ipld-prime/codec/dagcbor"
	"github.com/ipld/go-ipld-prime/node/basicnode"
	"github.com/multiformats/go-multibase"
	mh "github.com/multiformats/go-multihash"
	"math/big"
)

// Identity represents a basic identity structure.
type Identity struct {
	ID         string            // Unique ID for the identity
	PublicKey  string            // Hex representation of the public key
	PrivateKey *ecdsa.PrivateKey // ECDSA private key for signing
	Hash       string            // Hash of the identity (ID + PublicKey)
	Signatures map[string]string // Signatures for id and publicKey
	Bytes      []byte            // Encoded byte representation of the identity
	Type       string
}

// EncodedIdentity represents an Identity that has been encoded.
type EncodedIdentity struct {
	Identity Identity
	Bytes    []byte
	CID      cid.Cid
	Hash     string
}

// Sign generates a signature for the provided data using the private key.
func (i *Identity) Sign(data []byte) (string, error) {
	// Hash the data
	hashedData := sha256.Sum256(data)
	r, s, err := ecdsa.Sign(rand.Reader, i.PrivateKey, hashedData[:])
	if err != nil {
		return "", err
	}

	// Concatenate r and s values and encode as hex
	signature := append(r.Bytes(), s.Bytes()...)
	return hex.EncodeToString(signature), nil
}

// Verify checks if the provided signature is valid for the data.
func (i *Identity) Verify(signatureHex string, data []byte) bool {
	sigBytes, err := hex.DecodeString(signatureHex)
	if err != nil || len(sigBytes) < 64 {
		return false
	}

	r := new(big.Int).SetBytes(sigBytes[:len(sigBytes)/2])
	s := new(big.Int).SetBytes(sigBytes[len(sigBytes)/2:])

	// Hash the data
	hashedData := sha256.Sum256(data)

	// Verify the signature
	return ecdsa.Verify(&i.PrivateKey.PublicKey, hashedData[:], r, s)
}

// IsIdentity Checks if an identity has all required fields populated.
func IsIdentity(identity *Identity) bool {
	return identity != nil &&
		identity.ID != "" &&
		identity.Hash != "" &&
		identity.Bytes != nil &&
		identity.PublicKey != "" &&
		identity.Signatures != nil &&
		identity.Signatures["id"] != "" &&
		identity.Signatures["publicKey"] != "" &&
		identity.Type != ""
}

// IsEqual Checks if two identities are identical based on key properties.
func IsEqual(a, b *Identity) bool {
	if a == nil || b == nil {
		fmt.Println("One of the identities is nil.")
		return false
	}

	equal := true

	if a.ID != b.ID {
		fmt.Printf("IDs are not equal: a.ID = %v, b.ID = %v\n", a.ID, b.ID)
		equal = false
	}

	if a.PublicKey != b.PublicKey {
		fmt.Printf("Public keys are not equal: a.PublicKey = %v, b.PublicKey = %v\n", a.PublicKey, b.PublicKey)
		equal = false
	}
	if a.Signatures["id"] != b.Signatures["id"] {
		fmt.Printf("Signatures for 'id' are not equal: a.Signatures[\"id\"] = %v, b.Signatures[\"id\"] = %v\n", a.Signatures["id"], b.Signatures["id"])
		equal = false
	}
	if a.Signatures["publicKey"] != b.Signatures["publicKey"] {
		fmt.Printf("Signatures for 'publicKey' are not equal: a.Signatures[\"publicKey\"] = %v, b.Signatures[\"publicKey\"] = %v\n", a.Signatures["publicKey"], b.Signatures["publicKey"])
		equal = false
	}

	if a.Hash != b.Hash {
		fmt.Printf("Hashes are not equal: a.Hash = %v, b.Hash = %v\n", a.Hash, b.Hash)
		equal = false
	}

	return equal
}

// EncodeIdentity encodes an Identity instance into CBOR format and returns hash, bytes, and error.
func EncodeIdentity(identity Identity) (string, []byte, error) {
	// Initialize a basic map node for encoding with canonical field order
	nb := basicnode.Prototype__Map{}.NewBuilder()
	ma, _ := nb.BeginMap(4)

	// Assemble fields in a consistent order
	ma.AssembleKey().AssignString("id")
	ma.AssembleValue().AssignString(identity.ID)

	ma.AssembleKey().AssignString("publicKey")
	ma.AssembleValue().AssignString(identity.PublicKey)

	ma.AssembleKey().AssignString("signatures")
	sigMap, _ := ma.AssembleValue().BeginMap(int64(len(identity.Signatures)))
	for k, v := range identity.Signatures {
		sigMap.AssembleKey().AssignString(k)
		sigMap.AssembleValue().AssignString(v)
	}
	sigMap.Finish()

	ma.AssembleKey().AssignString("type")
	ma.AssembleValue().AssignString(identity.Type)

	ma.Finish()

	// Build the node and encode to CBOR
	node := nb.Build()
	var buf bytes.Buffer
	if err := dagcbor.Encode(node, &buf); err != nil {
		return "", nil, err
	}

	// Calculate CID for CBOR-encoded bytes
	hash, err := mh.Sum(buf.Bytes(), mh.SHA2_256, -1)
	if err != nil {
		return "", nil, err
	}
	c := cid.NewCidV1(cid.DagCBOR, hash)

	// Encode CID to base58btc for hash string
	hashStr, err := c.StringOfBase(multibase.Base58BTC)
	if err != nil {
		return "", nil, err
	}

	return hashStr, buf.Bytes(), nil
}

// DecodeIdentity decodes CBOR-encoded bytes back into an Identity struct.
func DecodeIdentity(encodedData []byte) (*Identity, error) {
	// Check if the encodedData is empty
	if len(encodedData) == 0 {
		return nil, errors.New("invalid or empty input data")
	}

	// Create a node for decoding
	nb := basicnode.Prototype__Map{}.NewBuilder()
	buf := bytes.NewReader(encodedData)

	// Decode CBOR data
	if err := dagcbor.Decode(nb, buf); err != nil {
		return nil, err
	}
	node := nb.Build()

	// Extract fields to reconstruct the Identity
	identity := Identity{}

	// Validate and set 'id' field
	if idNode, err := node.LookupByString("id"); err == nil {
		id, err := idNode.AsString()
		if err != nil || id == "" {
			return nil, errors.New("invalid or missing 'id' field")
		}
		identity.ID = id
	} else {
		return nil, errors.New("invalid or missing 'id' field")
	}

	// Validate and set 'publicKey' field
	if publicKeyNode, err := node.LookupByString("publicKey"); err == nil {
		publicKey, err := publicKeyNode.AsString()
		if err != nil || publicKey == "" {
			return nil, errors.New("invalid or missing 'publicKey' field")
		}
		identity.PublicKey = publicKey
	} else {
		return nil, errors.New("invalid or missing 'publicKey' field")
	}

	// Validate and set 'signatures' field as a map
	if sigNode, err := node.LookupByString("signatures"); err == nil {
		if sigNode.Kind() != ipld.Kind_Map {
			return nil, errors.New("invalid or missing 'signatures' field")
		}
		sigMap := make(map[string]string)

		// Iterate over map keys to ensure each signature is a string
		iter := sigNode.MapIterator()
		for !iter.Done() {
			keyNode, valueNode, _ := iter.Next()
			key, err := keyNode.AsString()
			if err != nil {
				return nil, errors.New("invalid key format in 'signatures'")
			}
			val, err := valueNode.AsString()
			if err != nil {
				return nil, errors.New("invalid signature format")
			}
			sigMap[key] = val
		}
		identity.Signatures = sigMap
	} else {
		return nil, errors.New("invalid or missing 'signatures' field")
	}

	// Validate and set 'type' field
	if typeNode, err := node.LookupByString("type"); err == nil {
		identityType, err := typeNode.AsString()
		if err != nil || identityType == "" {
			return nil, errors.New("invalid or missing 'type' field")
		}
		identity.Type = identityType
	} else {
		return nil, errors.New("invalid or missing 'type' field")
	}

	hash, encodedBytes, _ := EncodeIdentity(identity)
	identity.Hash = hash
	identity.Bytes = encodedBytes

	return &identity, nil
}
