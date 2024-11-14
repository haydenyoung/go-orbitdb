package providers

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/hex"
	"errors"
	"math/big"
	"orbitdb/go-orbitdb/identities/identitytypes"
	"orbitdb/go-orbitdb/keystore"
)

// PublicKeyProvider is a provider using public key-based identities and a KeyStore.
type PublicKeyProvider struct {
	keystore *keystore.KeyStore
}

// NewPublicKeyProvider creates a new PublicKeyProvider with a KeyStore.
func NewPublicKeyProvider(ks *keystore.KeyStore) *PublicKeyProvider {
	return &PublicKeyProvider{keystore: ks}
}

func (p *PublicKeyProvider) Type() string {
	return "publickey"
}

// CreateIdentity generates a new identity, signing the ID and public key.
func (p *PublicKeyProvider) CreateIdentity(id string) (*identitytypes.Identity, error) {
	// Check if a key already exists for this ID
	if !p.keystore.HasKey(id) {
		// If not, create a new key
		_, err := p.keystore.CreateKey(id)
		if err != nil {
			return nil, err
		}
	}

	privateKey, err := p.keystore.GetKey(id)
	if err != nil {
		return nil, err
	}

	// Generate the public key as a hex-encoded string
	publicKey := hex.EncodeToString(append(privateKey.PublicKey.X.Bytes(), privateKey.PublicKey.Y.Bytes()...))

	// Sign the ID and public key
	idSignature, err := p.keystore.SignMessage(id, []byte(id))
	if err != nil {
		return nil, err
	}

	publicKeySignature, err := p.keystore.SignMessage(id, []byte(publicKey))
	if err != nil {
		return nil, err
	}

	// Create the identity instance
	identity := &identitytypes.Identity{
		ID:        id,
		PublicKey: publicKey,
		Signatures: map[string]string{
			"id":        idSignature,
			"publicKey": publicKeySignature,
		},
		Type: p.Type(),
	}

	// Encode identity to generate hash and bytes representation
	hash, bytes, err := identitytypes.EncodeIdentity(*identity)
	if err != nil {
		return nil, err
	}
	identity.Hash = hash
	identity.Bytes = bytes

	return identity, nil
}

// VerifyIdentity checks and verifies the given identity, ensuring it has all required fields
// and that the signatures are valid.
func (p *PublicKeyProvider) VerifyIdentity(identity *identitytypes.Identity) (bool, error) {
	// Check that the identity has all necessary fields populated
	if !identitytypes.IsIdentity(identity) {
		return false, errors.New("identity is missing required fields")
	}

	// Decode the public key from the hex-encoded string
	publicKeyBytes, err := hex.DecodeString(identity.PublicKey)
	if err != nil || len(publicKeyBytes) < 64 {
		return false, errors.New("invalid public key encoding")
	}

	// Reconstruct the ecdsa.PublicKey
	pubKey := ecdsa.PublicKey{
		Curve: elliptic.P256(),
		X:     new(big.Int).SetBytes(publicKeyBytes[:len(publicKeyBytes)/2]),
		Y:     new(big.Int).SetBytes(publicKeyBytes[len(publicKeyBytes)/2:]),
	}

	// Verify the ID signature using the KeyStore's VerifyMessage method
	idVerified, err := p.keystore.VerifyMessage(pubKey, []byte(identity.ID), identity.Signatures["id"])
	if err != nil || !idVerified {
		return false, errors.New("invalid ID signature")
	}

	// Verify the public key signature using the KeyStore's VerifyMessage method
	publicKeyVerified, err := p.keystore.VerifyMessage(pubKey, []byte(identity.PublicKey), identity.Signatures["publicKey"])
	if err != nil || !publicKeyVerified {
		return false, errors.New("invalid public key signature")
	}

	return true, nil
}
