package identitytypes

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
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

// NewIdentity generates a new Identity instance with a random ECDSA key pair.
func NewIdentity(id string, privateKey *ecdsa.PrivateKey, idType string) (*Identity, error) {
	if id == "" {
		return nil, errors.New("identity ID is required")
	}
	if idType == "" {
		return nil, errors.New("identity type is required")
	}

	publicKey := hex.EncodeToString(append(privateKey.PublicKey.X.Bytes(), privateKey.PublicKey.Y.Bytes()...))

	identity := &Identity{
		ID:         id,
		PublicKey:  publicKey,
		PrivateKey: privateKey,
		Signatures: make(map[string]string),
		Type:       idType, // Set the type here
	}

	// Encode the identity to generate hash and bytes representation
	hash, bytes, err := EncodeIdentity(*identity)
	if err != nil {
		return nil, err
	}
	identity.Hash = hash
	identity.Bytes = bytes

	return identity, nil
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
