package jwtx

import (
	"crypto"
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// maxKeyIDLen bounds the "kid" header written by WithKeyID.
const maxKeyIDLen = 64

// SignOption configures SignWith.
type SignOption func(*signConfig)

type signConfig struct {
	kid    string
	hasKid bool
}

// WithKeyID writes kid as the "kid" header so that a verifier can select the
// key with VerifyWithKeySet. kid must be 1 to 64 characters from
// [A-Za-z0-9._-]; SignWith returns an error wrapping ErrTokenInvalid otherwise.
func WithKeyID(kid string) SignOption {
	return func(c *signConfig) { c.kid, c.hasKid = kid, true }
}

func validKeyID(kid string) bool {
	if kid == "" || len(kid) > maxKeyIDLen {
		return false
	}
	for i := 0; i < len(kid); i++ {
		c := kid[i]
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9',
			c == '.', c == '_', c == '-':
		default:
			return false
		}
	}
	return true
}

// Sign returns a serialized JWT for claims signed with key.
func Sign(claims Claims, key Key) (string, error) {
	return SignWith(claims, key)
}

// SignWith is Sign with options. Without options the output is identical to
// Sign.
func SignWith(claims Claims, key Key, opts ...SignOption) (string, error) {
	var cfg signConfig
	for _, opt := range opts {
		opt(&cfg)
	}
	header := map[string]string{
		"alg": string(key.Alg),
		"typ": "JWT",
	}
	if cfg.hasKid {
		if !validKeyID(cfg.kid) {
			return "", fmt.Errorf("%w: invalid key id", ErrTokenInvalid)
		}
		header["kid"] = cfg.kid
	}
	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", err
	}

	payload := buildPayload(claims)
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	signingInput := b64(headerJSON) + "." + b64(payloadJSON)
	signature, err := signRaw(key, []byte(signingInput))
	if err != nil {
		return "", err
	}
	return signingInput + "." + base64.RawURLEncoding.EncodeToString(signature), nil
}

func buildPayload(c Claims) map[string]any {
	p := make(map[string]any, len(c.Extra)+8)
	for k, v := range c.Extra {
		p[k] = v
	}
	if c.Subject != "" {
		p["sub"] = c.Subject
	}
	if c.Issuer != "" {
		p["iss"] = c.Issuer
	}
	if len(c.Audience) > 0 {
		p["aud"] = c.Audience
	}
	if !c.ExpiresAt.IsZero() {
		p["exp"] = c.ExpiresAt.Unix()
	}
	if !c.IssuedAt.IsZero() {
		p["iat"] = c.IssuedAt.Unix()
	}
	if !c.NotBefore.IsZero() {
		p["nbf"] = c.NotBefore.Unix()
	}
	if c.ID != "" {
		p["jti"] = c.ID
	}
	return p
}

func signRaw(key Key, msg []byte) ([]byte, error) {
	switch key.Alg {
	case AlgHS256:
		if len(key.HMAC) == 0 {
			return nil, ErrTokenInvalid
		}
		m := hmac.New(sha256.New, key.HMAC)
		m.Write(msg)
		return m.Sum(nil), nil
	case AlgHS512:
		if len(key.HMAC) == 0 {
			return nil, ErrTokenInvalid
		}
		m := hmac.New(sha512.New, key.HMAC)
		m.Write(msg)
		return m.Sum(nil), nil
	case AlgRS256:
		if key.RSAPriv == nil {
			return nil, fmt.Errorf("%w: RS256 sign requires a private key", ErrTokenInvalid)
		}
		h := sha256.Sum256(msg)
		return rsa.SignPKCS1v15(rand.Reader, key.RSAPriv, crypto.SHA256, h[:])
	default:
		return nil, ErrUnsupportedAlgo
	}
}

func b64(b []byte) string {
	return base64.RawURLEncoding.EncodeToString(b)
}

// nowFunc is overridable in tests.
var nowFunc = time.Now

// stripWhitespace is a defensive helper for token strings.
func stripWhitespace(s string) string {
	return strings.TrimSpace(s)
}
