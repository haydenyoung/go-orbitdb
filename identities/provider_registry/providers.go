package provider_registry

import (
	"errors"
	"orbitdb/go-orbitdb/identities"
)

// IdentityProvider defines the methods required for an identity provider
type IdentityProvider interface {
	GetID(identity *identities.Identity) (string, error)
	SignIdentity(data string, identity *identities.Identity) (string, error)
	VerifyIdentity(identity *identities.Identity) (bool, error)
	VerifyIdentityWithEntry(identity *identities.Identity, data []byte, signature string) (bool, error)
	Type() string
}

// Registry for storing providers by type
var identityProviders = make(map[string]IdentityProvider)

// UseIdentityProvider registers a new identity provider
func UseIdentityProvider(provider IdentityProvider) error {
	if provider.Type() == "" {
		return errors.New("identity provider must have a type")
	}
	if _, exists := identityProviders[provider.Type()]; exists {
		return errors.New("identity provider already registered")
	}
	identityProviders[provider.Type()] = provider
	return nil
}

// GetIdentityProvider retrieves an identity provider by type
func GetIdentityProvider(providerType string) (IdentityProvider, error) {
	provider, exists := identityProviders[providerType]
	if !exists {
		return nil, errors.New("identity provider not found")
	}
	return provider, nil
}
