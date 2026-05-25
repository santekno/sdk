package cryptox

import (
	"crypto/rand"
	"errors"
	"fmt"

	"golang.org/x/crypto/chacha20poly1305"
)

// ChaCha20KeyLen is the required key length for ChaCha20-Poly1305.
const ChaCha20KeyLen = 32

// EncryptChaCha20 encrypts plaintext using ChaCha20-Poly1305. Key MUST be
// exactly 32 bytes. The returned ciphertext has the nonce prepended.
func EncryptChaCha20(key, plaintext []byte) ([]byte, error) {
	if len(key) != ChaCha20KeyLen {
		return nil, fmt.Errorf("%w: ChaCha20-Poly1305 requires a %d-byte key", ErrInvalidKey, ChaCha20KeyLen)
	}
	aead, err := chacha20poly1305.New(key)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	ct := aead.Seal(nil, nonce, plaintext, nil)
	out := make([]byte, 0, len(nonce)+len(ct))
	out = append(out, nonce...)
	out = append(out, ct...)
	return out, nil
}

// DecryptChaCha20 decrypts a payload produced by EncryptChaCha20.
func DecryptChaCha20(key, ciphertext []byte) ([]byte, error) {
	if len(key) != ChaCha20KeyLen {
		return nil, fmt.Errorf("%w: ChaCha20-Poly1305 requires a %d-byte key", ErrInvalidKey, ChaCha20KeyLen)
	}
	aead, err := chacha20poly1305.New(key)
	if err != nil {
		return nil, err
	}
	if len(ciphertext) < aead.NonceSize() {
		return nil, errors.New("cryptox: ciphertext too short")
	}
	nonce, body := ciphertext[:aead.NonceSize()], ciphertext[aead.NonceSize():]
	return aead.Open(nil, nonce, body, nil)
}
