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

// Sign returns a serialized JWT for claims signed with key.
func Sign(claims Claims, key Key) (string, error) {
	header := map[string]string{
		"alg": string(key.Alg),
		"typ": "JWT",
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
