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
	"orbitdb/go-orbitdb/storage"
)

// Identities manages a collection of identities
type Identities struct {
	storage  map[string]*identitytypes.Identity
	provider Provider
	keystore *keystore.KeyStore
}

// NewIdentities initializes the identities manager with a specific provider and a KeyStore.
func NewIdentities(providerType string, storageBackend storage.Storage) (*Identities, error) {
	// Initialize a KeyStore instance
	ks := keystore.NewKeyStore(storageBackend)

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

// ClearAll clears all keys from the KeyStore
func (ids *Identities) ClearAll() {
	ids.keystore.Clear()
}

// AddManualKey allows adding an externally generated key.
func (ids *Identities) AddManualKey(id string, privateKey *ecdsa.PrivateKey) error {
	return ids.keystore.AddKey(id, privateKey)
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

// Sign signs the provided data using the identity's private key from the KeyStore.
func (ids *Identities) Sign(id string, data []byte) (string, error) {
	// Use KeyStore to sign the data
	return ids.keystore.SignMessage(id, data)
}

// Verify verifies the provided signature against the data and public key.
func (ids *Identities) Verify(signature string, identity *identitytypes.Identity, data []byte) bool {
	// Decode the public key from the identity's hex-encoded string
	publicKeyBytes, err := hex.DecodeString(identity.PublicKey)
	if err != nil || len(publicKeyBytes) < 64 {
		return false
	}

	// Reconstruct the ecdsa.PublicKey from the byte slice
	pubKey := ecdsa.PublicKey{
		Curve: elliptic.P256(),
		X:     new(big.Int).SetBytes(publicKeyBytes[:len(publicKeyBytes)/2]),
		Y:     new(big.Int).SetBytes(publicKeyBytes[len(publicKeyBytes)/2:]),
	}

	// Use VerifyMessage from KeyStore to verify the signature
	verified, err := ids.keystore.VerifyMessage(pubKey, data, signature)
	return err == nil && verified
}

// init registers the default provider.
func init() {
	lruStorage, _ := storage.NewLRUStorage(100)
	ks := keystore.NewKeyStore(lruStorage)
	RegisterProvider(providers.NewPublicKeyProvider(ks))
}
