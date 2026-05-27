package jwtx

import (
	"crypto"
	"crypto/hmac"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"fmt"
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
		return nil, fmt.Errorf("%w: header=%s, key=%s", ErrAlgorithmMismatch, header.Alg, key.Alg)
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
	default:
		return ErrUnsupportedAlgo
	}
}

func parseClaims(payload []byte) (*Claims, error) {
	var raw map[string]any
	if err := json.Unmarshal(payload, &raw); err != nil {
		return nil, ErrTokenMalformed
	}

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
			switch a := v.(type) {
			case string:
				c.Audience = []string{a}
			case []any:
				for _, x := range a {
					if s, ok := x.(string); ok {
						c.Audience = append(c.Audience, s)
					}
				}
			}
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

	now := nowFunc()
	if !c.ExpiresAt.IsZero() && now.After(c.ExpiresAt) {
		return nil, ErrTokenExpired
	}
	if !c.NotBefore.IsZero() && now.Before(c.NotBefore) {
		return nil, ErrTokenInvalid
	}
	return c, nil
}
