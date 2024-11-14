package identities

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/hex"
	"errors"
	"math/big"
	"orbitdb/go-orbitdb/identities/identitytypes"
	"orbitdb/go-orbitdb/identities/providers"
	"orbitdb/go-orbitdb/keystore"
)

// Identities manages a collection of identities
type Identities struct {
	storage  map[string]*identitytypes.Identity
	provider Provider
	keystore *keystore.KeyStore
}

// NewIdentities initializes the identities manager with a specific provider and a KeyStore.
func NewIdentities(providerType string) (*Identities, error) {
	// Initialize a KeyStore instance
	ks := keystore.NewKeyStore()

	// Select the provider based on providerType
	var provider Provider
	switch providerType {
	case "publickey":
		provider = providers.NewPublicKeyProvider(ks)
	default:
		return nil, errors.New("unsupported provider type")
	}

	return &Identities{
		storage:  make(map[string]*identitytypes.Identity),
		provider: provider,
		keystore: ks,
	}, nil
}

// CreateIdentity generates a new identity using the selected provider.
func (ids *Identities) CreateIdentity(id string) (*identitytypes.Identity, error) {
	identity, err := ids.provider.CreateIdentity(id)
	if err != nil {
		return nil, err
	}

	if !identitytypes.IsIdentity(identity) {
		return nil, errors.New("invalid identity created")
	}

	// Store the identity in the storage map
	ids.storage[identity.Hash] = identity
	return identity, nil
}

// VerifyIdentity verifies the provided identity.
func (ids *Identities) VerifyIdentity(identity *identitytypes.Identity) bool {
	verified, _ := ids.provider.VerifyIdentity(identity)
	return verified
}

// Sign signs the provided data using the identity's private key.
func (ids *Identities) Sign(identity *identitytypes.Identity, data []byte) (string, error) {
	if identity.PrivateKey == nil {
		return "", errors.New("private signing key not found for identity")
	}
	return identity.Sign(data)
}

// Verify verifies the provided signature against the data and public key.
func (ids *Identities) Verify(signature string, identity *identitytypes.Identity, data []byte) bool {
	return identity.Verify(signature, data)
}

// init registers the default provider.
func init() {
	ks := keystore.NewKeyStore()
	RegisterProvider(providers.NewPublicKeyProvider(ks))
}
