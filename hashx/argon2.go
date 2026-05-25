package hashx

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

// Argon2id default parameters (OWASP 2026).
const (
	Argon2Memory  = 64 * 1024 // KiB = 64 MiB
	Argon2Time    = 3
	Argon2Threads = 4
	Argon2SaltLen = 16
	Argon2KeyLen  = 32
)

// ErrEmptyPassword is returned by HashPassword when password is empty.
var ErrEmptyPassword = errors.New("hashx: password cannot be empty")

// ErrInvalidHash is returned when the supplied hash string is malformed.
var ErrInvalidHash = errors.New("hashx: invalid hash format")

// HashPassword hashes password using Argon2id with the OWASP-recommended
// parameters. The returned string is the standard $argon2id$ encoded format
// that contains all parameters plus the salt — safe to store as-is.
func HashPassword(password string) (string, error) {
	if password == "" {
		return "", ErrEmptyPassword
	}

	salt := make([]byte, Argon2SaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("hashx: read random salt: %w", err)
	}

	key := argon2.IDKey(
		[]byte(password),
		salt,
		Argon2Time,
		Argon2Memory,
		Argon2Threads,
		Argon2KeyLen,
	)

	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(key)

	return fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, Argon2Memory, Argon2Time, Argon2Threads, b64Salt, b64Hash,
	), nil
}

// VerifyPassword reports whether password matches the supplied Argon2id hash.
// Returns false on any decoding error or mismatch.
func VerifyPassword(password, hash string) bool {
	mem, time, threads, salt, want, err := decodeArgon2(hash)
	if err != nil {
		return false
	}

	keyLen := uint32(len(want)) // #nosec G115 -- argon2 key lengths are always small (≤64 bytes)
	got := argon2.IDKey([]byte(password), salt, time, mem, threads, keyLen)
	return subtle.ConstantTimeCompare(got, want) == 1
}

func decodeArgon2(hash string) (mem, time uint32, threads uint8, salt, key []byte, err error) {
	parts := strings.Split(hash, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		err = ErrInvalidHash
		return
	}
	var version int
	if _, err = fmt.Sscanf(parts[2], "v=%d", &version); err != nil {
		err = ErrInvalidHash
		return
	}
	if _, err = fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &mem, &time, &threads); err != nil {
		err = ErrInvalidHash
		return
	}
	if salt, err = base64.RawStdEncoding.DecodeString(parts[4]); err != nil {
		err = ErrInvalidHash
		return
	}
	if key, err = base64.RawStdEncoding.DecodeString(parts[5]); err != nil {
		err = ErrInvalidHash
		return
	}
	return mem, time, threads, salt, key, nil
}
