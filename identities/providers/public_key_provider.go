package providers

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/hex"
	"orbitdb/go-orbitdb/identities/identitytypes" // Updated import path
)

// PublicKeyProvider is a simple provider using public key-based identities.
type PublicKeyProvider struct{}

// Type returns the provider type.
func (p *PublicKeyProvider) Type() string {
	return "publickey"
}

// CreateIdentity generates a new Identity instance using the provided ECDSA private key.
func (p *PublicKeyProvider) CreateIdentity(id string, privateKey *ecdsa.PrivateKey) (*identitytypes.Identity, error) {
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
		Hash:       identityHash, // Set the Hash field here
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
