package identities

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/asn1"
	"encoding/hex"
	"math/big"
)

// Identity represents an identity with a public-private key pair
type Identity struct {
	ID         string
	PublicKey  ecdsa.PublicKey
	PrivateKey ecdsa.PrivateKey
	Identity   string
	Type       string // Type of identity, e.g., "publickey"
	Signatures struct {
		ID        string
		PublicKey string
	}
}

// ecdsaSignature is a struct for DER-encoded ECDSA signatures
type ecdsaSignature struct {
	R, S *big.Int
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
		ID:         identity,
		PublicKey:  publicKey,
		PrivateKey: *privateKey,
		Type:       "publickey",
		Signatures: struct{ ID, PublicKey string }{},
	}, nil
}

// Sign signs data with the identity's private key and returns the signature
func (id *Identity) Sign(data []byte) (string, error) {
	hash := sha256.Sum256(data)
	r, s, err := ecdsa.Sign(rand.Reader, &id.PrivateKey, hash[:])
	if err != nil {
		return "", err
	}

	// DER-encode the r and s values
	signature, err := asn1.Marshal(ecdsaSignature{r, s})
	if err != nil {
		return "", err
	}

	// Return the hex-encoded DER signature
	return hex.EncodeToString(signature), nil
}

// PublicKeyHex returns the public key as a hex-encoded string
func (id *Identity) PublicKeyHex() string {
	// Concatenate the X and Y coordinates of the public key, and encode to hex
	pubKeyBytes := append(id.PublicKey.X.Bytes(), id.PublicKey.Y.Bytes()...)
	return hex.EncodeToString(pubKeyBytes)
}
