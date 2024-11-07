package providers

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"encoding/asn1"
	"encoding/hex"
	"errors"
	"math/big"
	"orbitdb/go-orbitdb/identities"
)

type PublicKeyIdentityProvider struct {
}

// Struct for DER decoding
type ecdsaSignature struct {
	R, S *big.Int
}

// NewPublicKeyIdentityProvider initializes a new PublicKeyIdentityProvider
func NewPublicKeyIdentityProvider() *PublicKeyIdentityProvider {
	return &PublicKeyIdentityProvider{}
}

// Type returns the type of this provider
func (p *PublicKeyIdentityProvider) Type() string {
	return "publickey"
}

// GetID retrieves the identity ID as the public key in hex format
func (p *PublicKeyIdentityProvider) GetID(identity *identities.Identity) (string, error) {
	return identity.PublicKeyHex(), nil
}

// SignIdentity signs the given data using the identity’s private key
func (p *PublicKeyIdentityProvider) SignIdentity(data string, identity *identities.Identity) (string, error) {
	// Hash the data
	hash := sha256.Sum256([]byte(data))

	// Sign the hash using the private key
	r, s, err := ecdsa.Sign(rand.Reader, &identity.PrivateKey, hash[:])
	if err != nil {
		return "", err
	}

	// Convert r and s to hex and concatenate them
	return hex.EncodeToString(r.Bytes()) + hex.EncodeToString(s.Bytes()), nil
}

// VerifyIdentity verifies the signature of the identity itself
func (p *PublicKeyIdentityProvider) VerifyIdentity(identity *identities.Identity) (bool, error) {
	// Combine ID and Signatures.ID to form the data to verify
	data := identity.ID + identity.Signatures.ID

	// Decode the signature and public key
	signatureBytes, err := hex.DecodeString(identity.Signatures.PublicKey)
	if err != nil {
		return false, err
	}

	// Hash the data to verify
	hash := sha256.Sum256([]byte(data))

	// Verify the signature using the public key
	return verifyECDSASignature(&identity.PublicKey, hash[:], signatureBytes)
}

// VerifyIdentityWithEntry verifies an entry by checking the identity's public key and signature
func (p *PublicKeyIdentityProvider) VerifyIdentityWithEntry(identity *identities.Identity, data []byte, signature string) (bool, error) {
	// Hash the data to verify
	hash := sha256.Sum256(data)

	// Decode the hex-encoded DER signature
	signatureBytes, err := hex.DecodeString(signature)
	if err != nil {
		return false, errors.New("failed to decode signature from hex")
	}

	var sig ecdsaSignature
	if _, err := asn1.Unmarshal(signatureBytes, &sig); err != nil {
		return false, errors.New("invalid DER signature format")
	}

	// Verify the signature using the public key and the decoded r, s values
	return ecdsa.Verify(&identity.PublicKey, hash[:], sig.R, sig.S), nil
}

// Helper function to verify ECDSA signature
func verifyECDSASignature(publicKey *ecdsa.PublicKey, dataHash, signature []byte) (bool, error) {
	if len(signature) != 64 {
		return false, errors.New("invalid signature length")
	}

	r := new(big.Int).SetBytes(signature[:32])
	s := new(big.Int).SetBytes(signature[32:])

	return ecdsa.Verify(publicKey, dataHash, r, s), nil
}
