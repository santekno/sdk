package hashx

import (
	"crypto/hmac"
	"crypto/sha1" //nolint:gosec // HMAC-SHA1 is kept for protocol compatibility only
	"crypto/sha256"
	"crypto/sha512"
	"crypto/subtle"
)

// ConstantTimeEqual reports whether a and b are equal in constant time.
// Wraps crypto/subtle.ConstantTimeCompare.
func ConstantTimeEqual(a, b []byte) bool {
	return subtle.ConstantTimeCompare(a, b) == 1
}

// HMACSHA256 returns the HMAC-SHA-256 of data under key.
func HMACSHA256(key, data []byte) []byte {
	m := hmac.New(sha256.New, key)
	m.Write(data)
	return m.Sum(nil)
}

// HMACSHA512 returns the HMAC-SHA-512 of data under key.
func HMACSHA512(key, data []byte) []byte {
	m := hmac.New(sha512.New, key)
	m.Write(data)
	return m.Sum(nil)
}

// HMACSHA1 returns the HMAC-SHA-1 of data under key.
//
// It exists only for protocol compatibility (for example the X-Hub-Signature
// header of WebSub, which mandates sha1). SHA-1 must not be used in new
// designs: use HMACSHA256 or HMACSHA512 instead. Callers verifying a received
// signature must compare it with ConstantTimeEqual (or hmac.Equal), never with
// bytes.Equal or ==.
func HMACSHA1(key, data []byte) []byte {
	m := hmac.New(sha1.New, key)
	m.Write(data)
	return m.Sum(nil)
}
