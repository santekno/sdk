package jwtx

import (
	"crypto/ecdsa"
	"crypto/rsa"
	"errors"
	"time"
)

// Algorithm identifies a supported JWT signing algorithm.
type Algorithm string

// Supported algorithms.
const (
	AlgHS256 Algorithm = "HS256"
	AlgHS512 Algorithm = "HS512"
	AlgRS256 Algorithm = "RS256"
	AlgES256 Algorithm = "ES256"
)

// Errors returned by Sign and Verify.
var (
	ErrTokenExpired      = errors.New("jwtx: token expired")
	ErrTokenInvalid      = errors.New("jwtx: token invalid")
	ErrTokenMalformed    = errors.New("jwtx: token malformed")
	ErrAlgorithmNone     = errors.New("jwtx: alg=none is forbidden")
	ErrAlgorithmMismatch = errors.New("jwtx: token algorithm does not match key")
	ErrSignatureInvalid  = errors.New("jwtx: invalid signature")
	ErrUnsupportedAlgo   = errors.New("jwtx: unsupported algorithm")
)

// Errors returned by VerifyWithKeySet, ParseKeySet, and JWKSClient.
var (
	// ErrAlgorithmNotAllowed reports a token alg outside the caller's allow-list.
	ErrAlgorithmNotAllowed = errors.New("jwtx: token algorithm is not allowed")
	// ErrKeyIDMissing reports a token without a "kid" header.
	ErrKeyIDMissing = errors.New("jwtx: token has no key id")
	// ErrKeyNotFound reports a "kid" the key set does not hold.
	ErrKeyNotFound = errors.New("jwtx: key not found")
	// ErrKeySetUnavailable reports that no key could be loaded at all. Unlike
	// ErrKeyNotFound it says nothing about the token: callers should answer
	// with a server-side failure rather than an authentication failure.
	ErrKeySetUnavailable = errors.New("jwtx: key set unavailable")
	// ErrJWKSInvalid reports a JWKS document that cannot be used.
	ErrJWKSInvalid = errors.New("jwtx: invalid JWKS")
	// ErrJWKSURLInvalid reports a JWKS URL the client refuses to fetch.
	ErrJWKSURLInvalid = errors.New("jwtx: JWKS URL must be https")
)

// Claims is the JWT payload. Custom claims may be added via Extra.
type Claims struct {
	Subject   string         `json:"sub,omitempty"`
	Issuer    string         `json:"iss,omitempty"`
	Audience  []string       `json:"aud,omitempty"`
	ExpiresAt time.Time      `json:"-"`
	IssuedAt  time.Time      `json:"-"`
	NotBefore time.Time      `json:"-"`
	ID        string         `json:"jti,omitempty"`
	Extra     map[string]any `json:"-"`
}

// Key is the signing/verification key bound to a specific algorithm.
type Key struct {
	Alg     Algorithm
	HMAC    []byte
	RSAPub  *rsa.PublicKey
	RSAPriv *rsa.PrivateKey
	ECPub   *ecdsa.PublicKey
}

// HS256 returns a symmetric key for HS256.
func HS256(secret []byte) Key { return Key{Alg: AlgHS256, HMAC: secret} }

// HS512 returns a symmetric key for HS512.
func HS512(secret []byte) Key { return Key{Alg: AlgHS512, HMAC: secret} }

// RS256Sign returns an RS256 signing key from a private key.
func RS256Sign(priv *rsa.PrivateKey) Key { return Key{Alg: AlgRS256, RSAPriv: priv} }

// RS256Verify returns an RS256 verification key from a public key.
func RS256Verify(pub *rsa.PublicKey) Key { return Key{Alg: AlgRS256, RSAPub: pub} }

// ES256Verify returns an ES256 verification key from a P-256 public key.
// Signing with ES256 is not supported.
func ES256Verify(pub *ecdsa.PublicKey) Key { return Key{Alg: AlgES256, ECPub: pub} }
