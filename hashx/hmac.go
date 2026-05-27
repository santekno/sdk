package hashx

import (
	"crypto/hmac"
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
