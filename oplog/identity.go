package oplog

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"math/big"
)

// Identity represents an identity with a public-private key pair
type Identity struct {
	PublicKey  ecdsa.PublicKey
	PrivateKey ecdsa.PrivateKey
	Identity   string
}

// NewIdentity generates a new identity with a public-private key pair
func NewIdentity() (*Identity, error) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}

	publicKey := privateKey.PublicKey
	identityHash := sha256.Sum256(append(publicKey.X.Bytes(), publicKey.Y.Bytes()...))
	identity := hex.EncodeToString(identityHash[:])

	return &Identity{
		PublicKey:  publicKey,
		PrivateKey: *privateKey,
		Identity:   identity,
	}, nil
}

// Sign signs data with the identity's private key and returns the signature
func (id *Identity) Sign(data []byte) (string, error) {
	hash := sha256.Sum256(data)
	r, s, err := ecdsa.Sign(rand.Reader, &id.PrivateKey, hash[:])
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(r.Bytes()) + hex.EncodeToString(s.Bytes()), nil
}

// VerifySignature verifies the signature for the given data
func (id *Identity) VerifySignature(data []byte, signature string) (bool, error) {
	hash := sha256.Sum256(data)
	r := new(big.Int)
	s := new(big.Int)

	sigLen := len(signature) / 2
	r.SetString(signature[:sigLen], 16)
	s.SetString(signature[sigLen:], 16)

	valid := ecdsa.Verify(&id.PublicKey, hash[:], r, s)
	return valid, nil
}

// PublicKeyHex returns the public key as a hex-encoded string
func (id *Identity) PublicKeyHex() string {
	// Concatenate the X and Y coordinates of the public key, and encode to hex
	pubKeyBytes := append(id.PublicKey.X.Bytes(), id.PublicKey.Y.Bytes()...)
	return hex.EncodeToString(pubKeyBytes)
}
