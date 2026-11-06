package jwtx

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/hmac"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"time"
)

// Verify parses, validates, and returns the claims of a JWT string signed
// with key. The signing algorithm is taken from the token header but MUST
// match key.Alg. The "alg=none" token is always rejected.
func Verify(token string, key Key) (*Claims, error) {
	token = stripWhitespace(token)
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, ErrTokenMalformed
	}

	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, ErrTokenMalformed
	}
	var header struct {
		Alg string `json:"alg"`
		Typ string `json:"typ"`
	}
	if err := json.Unmarshal(headerBytes, &header); err != nil {
		return nil, ErrTokenMalformed
	}

	if strings.EqualFold(header.Alg, "none") || header.Alg == "" {
		return nil, ErrAlgorithmNone
	}
	if !strings.EqualFold(header.Alg, string(key.Alg)) {
		return nil, ErrAlgorithmMismatch
	}

	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return nil, ErrTokenMalformed
	}
	signingInput := []byte(parts[0] + "." + parts[1])
	if err := verifyRaw(key, signingInput, signature); err != nil {
		return nil, err
	}

	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, ErrTokenMalformed
	}
	return parseClaims(payloadBytes)
}

func verifyRaw(key Key, msg, sig []byte) error {
	switch key.Alg {
	case AlgHS256:
		m := hmac.New(sha256.New, key.HMAC)
		m.Write(msg)
		expected := m.Sum(nil)
		if subtle.ConstantTimeCompare(sig, expected) != 1 {
			return ErrSignatureInvalid
		}
		return nil
	case AlgHS512:
		m := hmac.New(sha512.New, key.HMAC)
		m.Write(msg)
		expected := m.Sum(nil)
		if subtle.ConstantTimeCompare(sig, expected) != 1 {
			return ErrSignatureInvalid
		}
		return nil
	case AlgRS256:
		if key.RSAPub == nil {
			return fmt.Errorf("%w: RS256 verify requires a public key", ErrTokenInvalid)
		}
		h := sha256.Sum256(msg)
		if err := rsa.VerifyPKCS1v15(key.RSAPub, crypto.SHA256, h[:], sig); err != nil {
			return ErrSignatureInvalid
		}
		return nil
	case AlgES256:
		return verifyES256(key.ECPub, msg, sig)
	default:
		return ErrUnsupportedAlgo
	}
}

// es256SignatureSize is the JWS encoding of an ES256 signature: r and s as
// fixed-width big-endian integers (RFC 7518 section 3.4). ASN.1 DER is invalid.
const es256SignatureSize = 64

func verifyES256(pub *ecdsa.PublicKey, msg, sig []byte) error {
	// crypto/ecdsa dereferences all three fields of the key.
	if pub == nil || pub.Curve != elliptic.P256() || pub.X == nil || pub.Y == nil {
		return fmt.Errorf("%w: ES256 verify requires a P-256 public key", ErrTokenInvalid)
	}
	if len(sig) != es256SignatureSize {
		return ErrSignatureInvalid
	}
	h := sha256.Sum256(msg)
	r := new(big.Int).SetBytes(sig[:es256SignatureSize/2])
	s := new(big.Int).SetBytes(sig[es256SignatureSize/2:])
	if !ecdsa.Verify(pub, h[:], r, s) {
		return ErrSignatureInvalid
	}
	return nil
}

func parseClaims(payload []byte) (*Claims, error) {
	var raw map[string]any
	if err := json.Unmarshal(payload, &raw); err != nil {
		return nil, ErrTokenMalformed
	}
	c := claimsFromMap(raw)

	now := nowFunc()
	if !c.ExpiresAt.IsZero() && now.After(c.ExpiresAt) {
		return nil, ErrTokenExpired
	}
	if !c.NotBefore.IsZero() && now.Before(c.NotBefore) {
		return nil, ErrTokenInvalid
	}
	return c, nil
}

func claimsFromMap(raw map[string]any) *Claims {
	c := &Claims{Extra: make(map[string]any)}
	for k, v := range raw {
		switch k {
		case "sub":
			c.Subject, _ = v.(string)
		case "iss":
			c.Issuer, _ = v.(string)
		case "jti":
			c.ID, _ = v.(string)
		case "aud":
			c.Audience = audienceFrom(v)
		case "exp":
			if f, ok := v.(float64); ok {
				c.ExpiresAt = time.Unix(int64(f), 0)
			}
		case "iat":
			if f, ok := v.(float64); ok {
				c.IssuedAt = time.Unix(int64(f), 0)
			}
		case "nbf":
			if f, ok := v.(float64); ok {
				c.NotBefore = time.Unix(int64(f), 0)
			}
		default:
			c.Extra[k] = v
		}
	}
	return c
}

func audienceFrom(v any) []string {
	switch a := v.(type) {
	case string:
		return []string{a}
	case []any:
		var aud []string
		for _, x := range a {
			if s, ok := x.(string); ok {
				aud = append(aud, s)
			}
		}
		return aud
	default:
		return nil
	}
}
