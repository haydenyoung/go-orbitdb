package providers

import "orbitdb/go-orbitdb/identities/provider_registry"

func init() {
	// Register the PublicKeyIdentityProvider
	provider := NewPublicKeyIdentityProvider()
	provider_registry.UseIdentityProvider(provider)
}
