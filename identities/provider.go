package identities

import (
	"crypto/ecdsa"
	"errors"
	"orbitdb/go-orbitdb/identities/identitytypes"
)

// Provider defines an interface for identity providers.
type Provider interface {
	Type() string
	CreateIdentity(id string, privateKey *ecdsa.PrivateKey) (*identitytypes.Identity, error)
	VerifyIdentity(identity *identitytypes.Identity, signature string, data []byte) bool
}

// providerRegistry stores available providers.
var providerRegistry = make(map[string]Provider)

// RegisterProvider registers a new provider for creating identities.
func RegisterProvider(provider Provider) {
	providerRegistry[provider.Type()] = provider
}

// GetProvider retrieves a provider by type.
func GetProvider(providerType string) (Provider, error) {
	provider, exists := providerRegistry[providerType]
	if !exists {
		return nil, errors.New("provider not found")
	}
	return provider, nil
}
