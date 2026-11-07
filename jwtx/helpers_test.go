package jwtx_test

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"sync"
	"testing"
	"time"
)

const testKID = "key-1"

var (
	testECOnce sync.Once
	testEC     *ecdsa.PrivateKey

	testRSAOnce sync.Once
	testRSA     *rsa.PrivateKey
)

func ecKey(tb testing.TB) *ecdsa.PrivateKey {
	tb.Helper()
	testECOnce.Do(func() {
		var err error
		if testEC, err = ecdsa.GenerateKey(elliptic.P256(), rand.Reader); err != nil {
			tb.Fatalf("GenerateKey: %v", err)
		}
	})
	return testEC
}

func rsaKey(tb testing.TB) *rsa.PrivateKey {
	tb.Helper()
	testRSAOnce.Do(func() {
		var err error
		if testRSA, err = rsa.GenerateKey(rand.Reader, 2048); err != nil {
			tb.Fatalf("GenerateKey: %v", err)
		}
	})
	return testRSA
}

func b64(data []byte) string { return base64.RawURLEncoding.EncodeToString(data) }

func b64JSON(tb testing.TB, v any) string {
	tb.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		tb.Fatalf("Marshal: %v", err)
	}
	return b64(data)
}

// signingInput returns the first two segments of a token.
func signingInput(tb testing.TB, header, claims map[string]any) string {
	tb.Helper()
	return b64JSON(tb, header) + "." + b64JSON(tb, claims)
}

// validClaims returns claims that pass every time check.
func validClaims() map[string]any {
	now := time.Now()
	return map[string]any{
		"sub": "user-1",
		"iss": "https://issuer.test/auth/v1",
		"aud": "authenticated",
		"iat": now.Add(-time.Minute).Unix(),
		"exp": now.Add(time.Hour).Unix(),
		"app_metadata": map[string]any{
			"role": "editor",
		},
	}
}

func es256Header(kid string) map[string]any {
	return map[string]any{"alg": "ES256", "typ": "JWT", "kid": kid}
}

// rawES256 signs input and returns the JWS r||s encoding.
func rawES256(tb testing.TB, priv *ecdsa.PrivateKey, input string) []byte {
	tb.Helper()
	h := sha256.Sum256([]byte(input))
	r, s, err := ecdsa.Sign(rand.Reader, priv, h[:])
	if err != nil {
		tb.Fatalf("ecdsa.Sign: %v", err)
	}
	sig := make([]byte, 64)
	r.FillBytes(sig[:32])
	s.FillBytes(sig[32:])
	return sig
}

func signES256(tb testing.TB, priv *ecdsa.PrivateKey, header, claims map[string]any) string {
	tb.Helper()
	input := signingInput(tb, header, claims)
	return input + "." + b64(rawES256(tb, priv, input))
}

func signRS256(tb testing.TB, priv *rsa.PrivateKey, header, claims map[string]any) string {
	tb.Helper()
	input := signingInput(tb, header, claims)
	h := sha256.Sum256([]byte(input))
	sig, err := rsa.SignPKCS1v15(rand.Reader, priv, crypto.SHA256, h[:])
	if err != nil {
		tb.Fatalf("SignPKCS1v15: %v", err)
	}
	return input + "." + b64(sig)
}

func signHS256(tb testing.TB, secret []byte, header, claims map[string]any) string {
	tb.Helper()
	input := signingInput(tb, header, claims)
	m := hmac.New(sha256.New, secret)
	m.Write([]byte(input))
	return input + "." + b64(m.Sum(nil))
}

func ecJWK(tb testing.TB, pub *ecdsa.PublicKey, kid string) map[string]any {
	tb.Helper()
	point, err := pub.Bytes()
	if err != nil {
		tb.Fatalf("PublicKey.Bytes: %v", err)
	}
	return map[string]any{
		"kty": "EC", "crv": "P-256", "kid": kid, "use": "sig", "alg": "ES256",
		"x": b64(point[1:33]), "y": b64(point[33:]),
	}
}

func rsaJWK(pub *rsa.PublicKey, kid string) map[string]any {
	return map[string]any{
		"kty": "RSA", "kid": kid, "use": "sig", "alg": "RS256",
		"n": b64(pub.N.Bytes()), "e": "AQAB",
	}
}

func jwksDoc(tb testing.TB, keys ...map[string]any) []byte {
	tb.Helper()
	data, err := json.Marshal(map[string]any{"keys": keys})
	if err != nil {
		tb.Fatalf("Marshal: %v", err)
	}
	return data
}
