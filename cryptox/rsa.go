package cryptox

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
)

// EncryptRSAOAEP encrypts plaintext using RSA-OAEP with SHA-256.
func EncryptRSAOAEP(pub *rsa.PublicKey, plaintext []byte) ([]byte, error) {
	return rsa.EncryptOAEP(sha256.New(), rand.Reader, pub, plaintext, nil)
}

// DecryptRSAOAEP decrypts ciphertext produced by EncryptRSAOAEP.
func DecryptRSAOAEP(priv *rsa.PrivateKey, ciphertext []byte) ([]byte, error) {
	return rsa.DecryptOAEP(sha256.New(), rand.Reader, priv, ciphertext, nil)
}

// GenerateRSAKeyPair returns a fresh RSA private key of the given bit size.
// 4096 bits is recommended for new keys; 2048 is acceptable for legacy interop.
func GenerateRSAKeyPair(bits int) (*rsa.PrivateKey, error) {
	return rsa.GenerateKey(rand.Reader, bits)
}
