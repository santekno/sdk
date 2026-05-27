package apikey

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
)

// RandomLen is the byte length of the random portion of generated keys.
// 24 bytes = 192 bits of entropy.
const RandomLen = 24

// ErrEmptyPrefix is returned by Generate when prefix is empty.
var ErrEmptyPrefix = errors.New("apikey: prefix cannot be empty")

// Generate returns a fresh API key in the form "{prefix}_{random}" along with
// the SHA-256 hash of the full plaintext key for at-rest storage. The plaintext
// MUST be shown to the user only once.
func Generate(prefix string) (plaintext, hash string, err error) {
	if prefix == "" {
		return "", "", ErrEmptyPrefix
	}
	random := make([]byte, RandomLen)
	if _, err := rand.Read(random); err != nil {
		return "", "", fmt.Errorf("apikey: read random: %w", err)
	}
	encoded := base64.RawURLEncoding.EncodeToString(random)
	plaintext = prefix + "_" + encoded
	hash = HashKey(plaintext)
	return plaintext, hash, nil
}

// HashKey returns the hex-encoded SHA-256 hash of plaintext.
// Use this to store the hash at rest and to look up a key by its hash.
func HashKey(plaintext string) string {
	sum := sha256.Sum256([]byte(plaintext))
	return hex.EncodeToString(sum[:])
}

// Verify reports whether plaintext matches the supplied at-rest hash.
// Comparison is constant-time.
func Verify(plaintext, hash string) bool {
	got := HashKey(plaintext)
	return subtle.ConstantTimeCompare([]byte(got), []byte(hash)) == 1
}

// Prefix returns the prefix portion of plaintext (everything before the
// first "_"), or "" if plaintext does not contain an underscore.
func Prefix(plaintext string) string {
	idx := strings.Index(plaintext, "_")
	if idx == -1 {
		return ""
	}
	return plaintext[:idx]
}
