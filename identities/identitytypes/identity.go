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
}

// NewIdentity generates a new Identity instance with a random ECDSA key pair.
func NewIdentity(id string, privateKey *ecdsa.PrivateKey) (*Identity, error) {
	if id == "" {
		return nil, errors.New("identity ID is required")
	}
	publicKey := hex.EncodeToString(append(privateKey.PublicKey.X.Bytes(), privateKey.PublicKey.Y.Bytes()...))

	hash := sha256.Sum256([]byte(id + publicKey))
	return &Identity{
		ID:         id,
		PublicKey:  publicKey,
		PrivateKey: privateKey,
		Hash:       hex.EncodeToString(hash[:]),
	}, nil
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
