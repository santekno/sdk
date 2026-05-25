// Package cryptox provides authenticated encryption, signing, and key
// generation helpers built on Go's stdlib crypto packages and
// golang.org/x/crypto.
//
// Only secure, modern algorithms are supported:
//
//   - AES-256-GCM and ChaCha20-Poly1305 (authenticated encryption)
//   - RSA-OAEP with SHA-256 (asymmetric encryption)
//   - Ed25519 (signing)
//
// Deprecated algorithms (MD5, SHA-1, DES, RC4, 3DES, ECB) are intentionally
// absent.
package cryptox
