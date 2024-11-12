package providers

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/sha256"
	"encoding/hex"
	"math/big"
	"orbitdb/go-orbitdb/identities/identitytypes"
)

// PublicKeyProvider is a simple provider using public key-based identities.
type PublicKeyProvider struct{}

// Type returns the provider type.
func (p *PublicKeyProvider) Type() string {
	return "publickey"
}

// createHardcodedKeyPair creates a fixed ECDSA private key for hardcoded testing.
func createHardcodedKeyPair() *ecdsa.PrivateKey {
	privateKey := new(ecdsa.PrivateKey)
	privateKey.PublicKey.Curve = elliptic.P256()

	privateKey.D, _ = new(big.Int).SetString("5e5d9e0a44685aee2282a44d2d3e9a1b", 16)
	privateKey.PublicKey.X, privateKey.PublicKey.Y = privateKey.PublicKey.Curve.ScalarBaseMult(privateKey.D.Bytes())

	return privateKey
}

// CreateIdentity generates a new Identity instance using the hardcoded ECDSA private key.
func (p *PublicKeyProvider) CreateIdentity(id string) (*identitytypes.Identity, error) {
	privateKey := createHardcodedKeyPair()

	// Convert the public key to a hex string
	publicKey := hex.EncodeToString(append(privateKey.PublicKey.X.Bytes(), privateKey.PublicKey.Y.Bytes()...))

	// Hash the ID and public key to create a unique identity hash
	hash := sha256.Sum256([]byte(id + publicKey))
	identityHash := hex.EncodeToString(hash[:])

	// Create and return the identity with the calculated hash
	return &identitytypes.Identity{
		ID:         id,
		PublicKey:  publicKey,
		PrivateKey: privateKey,
		Hash:       identityHash,
	}, nil
}

// VerifyIdentity verifies the identity signature.
func (p *PublicKeyProvider) VerifyIdentity(identity *identitytypes.Identity, signature string, data []byte) bool {
	return identity.Verify(signature, data)
}

// NewPublicKeyProvider creates a new instance of PublicKeyProvider.
func NewPublicKeyProvider() *PublicKeyProvider {
	return &PublicKeyProvider{}
}
