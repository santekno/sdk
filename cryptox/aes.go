package cryptox

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"
)

// AESKeyLen is the required key length for AES-256.
const AESKeyLen = 32

// ErrInvalidKey is returned when a key has an unsupported length.
var ErrInvalidKey = errors.New("cryptox: invalid key length")

// EncryptAESGCM encrypts plaintext using AES-256-GCM. Key MUST be exactly
// 32 bytes. The returned ciphertext has the nonce prepended (12 bytes nonce
// + ciphertext + 16-byte auth tag).
func EncryptAESGCM(key, plaintext []byte) ([]byte, error) {
	if len(key) != AESKeyLen {
		return nil, fmt.Errorf("%w: AES-256-GCM requires a %d-byte key", ErrInvalidKey, AESKeyLen)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	ct := gcm.Seal(nil, nonce, plaintext, nil)
	out := make([]byte, 0, len(nonce)+len(ct))
	out = append(out, nonce...)
	out = append(out, ct...)
	return out, nil
}

// DecryptAESGCM decrypts a payload produced by EncryptAESGCM.
func DecryptAESGCM(key, ciphertext []byte) ([]byte, error) {
	if len(key) != AESKeyLen {
		return nil, fmt.Errorf("%w: AES-256-GCM requires a %d-byte key", ErrInvalidKey, AESKeyLen)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	if len(ciphertext) < gcm.NonceSize() {
		return nil, errors.New("cryptox: ciphertext too short")
	}
	nonce, body := ciphertext[:gcm.NonceSize()], ciphertext[gcm.NonceSize():]
	return gcm.Open(nil, nonce, body, nil)
}

// GenerateAESKey returns a fresh random 32-byte key suitable for AES-256.
func GenerateAESKey() ([]byte, error) {
	k := make([]byte, AESKeyLen)
	if _, err := rand.Read(k); err != nil {
		return nil, err
	}
	return k, nil
}
