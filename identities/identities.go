package identities

import (
	"crypto/ecdsa"
	"orbitdb/go-orbitdb/identities/identitytypes" // Updated import path
	"orbitdb/go-orbitdb/identities/providers"
)

// Identities manages a collection of identities
type Identities struct {
	storage  map[string]*identitytypes.Identity
	provider Provider
}

// NewIdentities initializes the identities manager with a specific provider.
func NewIdentities(providerType string) (*Identities, error) {
	provider, err := GetProvider(providerType)
	if err != nil {
		return nil, err
	}

	return &Identities{
		storage:  make(map[string]*identitytypes.Identity),
		provider: provider,
	}, nil
}

// CreateIdentity generates a new identity using the selected provider.
func (ids *Identities) CreateIdentity(id string, privateKey *ecdsa.PrivateKey) (*identitytypes.Identity, error) {
	identity, err := ids.provider.CreateIdentity(id, privateKey)
	if err != nil {
		return nil, err
	}

	// Store the identity in the storage map
	ids.storage[identity.Hash] = identity
	return identity, nil
}

// VerifyIdentity verifies the authenticity of the provided identity.
func (ids *Identities) VerifyIdentity(identity *identitytypes.Identity, signature string, data []byte) bool {
	return ids.provider.VerifyIdentity(identity, signature, data)
}

// init registers the default provider.
func init() {
	RegisterProvider(providers.NewPublicKeyProvider())
}
