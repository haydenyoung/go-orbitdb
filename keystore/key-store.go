package keystore

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"math/big"
	"sync"
)

// KeyStore provides a simple key management system.
type KeyStore struct {
	storage map[string]*ecdsa.PrivateKey
	mu      sync.Mutex
}

// NewKeyStore initializes a new in-memory KeyStore.
func NewKeyStore() *KeyStore {
	return &KeyStore{
		storage: make(map[string]*ecdsa.PrivateKey),
	}
}

// CreateKey generates a new ECDSA key pair and stores it under the given ID.
func (ks *KeyStore) CreateKey(id string) (*ecdsa.PrivateKey, error) {
	ks.mu.Lock()
	defer ks.mu.Unlock()

	if _, exists := ks.storage[id]; exists {
		return nil, errors.New("key already exists for this ID")
	}

	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}

	ks.storage[id] = privateKey
	return privateKey, nil
}

// HasKey checks if a key exists for a given ID.
func (ks *KeyStore) HasKey(id string) bool {
	ks.mu.Lock()
	defer ks.mu.Unlock()

	_, exists := ks.storage[id]
	return exists
}

// AddKey Adds a private key directly to the keystore (e.g., for imported keys).
func (ks *KeyStore) AddKey(id string, privateKey *ecdsa.PrivateKey) error {
	ks.mu.Lock()
	defer ks.mu.Unlock()

	if _, exists := ks.storage[id]; exists {
		return errors.New("key already exists for this ID")
	}
	ks.storage[id] = privateKey
	return nil
}

// Clear removes all keys from the KeyStore.
func (ks *KeyStore) Clear() {
	ks.mu.Lock()
	defer ks.mu.Unlock()

	ks.storage = make(map[string]*ecdsa.PrivateKey)
}

// GetKey retrieves a private key by ID from storage.
func (ks *KeyStore) GetKey(id string) (*ecdsa.PrivateKey, error) {
	ks.mu.Lock()
	defer ks.mu.Unlock()

	privateKey, exists := ks.storage[id]
	if !exists {
		return nil, errors.New("key not found")
	}

	return privateKey, nil
}

// SignMessage signs data using the private key associated with the given ID.
func (ks *KeyStore) SignMessage(id string, data []byte) (string, error) {
	privateKey, err := ks.GetKey(id)
	if err != nil {
		return "", err
	}

	hash := sha256.Sum256(data)
	r, s, err := ecdsa.Sign(rand.Reader, privateKey, hash[:])
	if err != nil {
		return "", err
	}

	signature := append(r.Bytes(), s.Bytes()...)
	return hex.EncodeToString(signature), nil
}

// VerifyMessage verifies the signature against the data using the public key.
func (ks *KeyStore) VerifyMessage(publicKey ecdsa.PublicKey, data []byte, signature string) (bool, error) {
	sigBytes, err := hex.DecodeString(signature)
	if err != nil || len(sigBytes) < 64 {
		return false, err
	}

	r := new(big.Int).SetBytes(sigBytes[:len(sigBytes)/2])
	s := new(big.Int).SetBytes(sigBytes[len(sigBytes)/2:])

	hash := sha256.Sum256(data)
	return ecdsa.Verify(&publicKey, hash[:], r, s), nil
}
