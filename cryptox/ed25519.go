package cryptox

import (
	"crypto/ed25519"
	"crypto/rand"
)

// SignEd25519 signs msg with priv using Ed25519. Returns a 64-byte signature.
func SignEd25519(priv ed25519.PrivateKey, msg []byte) []byte {
	return ed25519.Sign(priv, msg)
}

// VerifyEd25519 reports whether sig is a valid Ed25519 signature of msg under pub.
func VerifyEd25519(pub ed25519.PublicKey, msg, sig []byte) bool {
	return ed25519.Verify(pub, msg, sig)
}

// GenerateEd25519KeyPair returns a fresh Ed25519 key pair.
func GenerateEd25519KeyPair() (ed25519.PublicKey, ed25519.PrivateKey, error) {
	return ed25519.GenerateKey(rand.Reader)
}
