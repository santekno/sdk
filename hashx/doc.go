// Package hashx provides password hashing and HMAC helpers.
//
// The default password hash is Argon2id with OWASP-recommended 2026 parameters
// (m=64 MiB, t=3, p=4, salt=16 B, key=32 B). bcrypt is provided for legacy
// interoperability. All comparisons use constant-time equality.
//
//	hash, _ := hashx.HashPassword("my-password")
//	ok := hashx.VerifyPassword("my-password", hash)
package hashx
