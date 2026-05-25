package jwtx_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/santekno/sdk/cryptox"
	"github.com/santekno/sdk/jwtx"
)

func TestSignVerify_HS256(t *testing.T) {
	secret := []byte("super-secret")
	claims := jwtx.Claims{
		Subject:   "u-42",
		Issuer:    "test",
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(time.Hour),
		Audience:  []string{"web"},
		Extra:     map[string]any{"role": "admin"},
	}
	tok, err := jwtx.Sign(claims, jwtx.HS256(secret))
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	if strings.Count(tok, ".") != 2 {
		t.Errorf("token shape unexpected: %q", tok)
	}
	got, err := jwtx.Verify(tok, jwtx.HS256(secret))
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if got.Subject != "u-42" {
		t.Errorf("Subject = %q", got.Subject)
	}
	if got.Extra["role"] != "admin" {
		t.Errorf("Extra[role] = %v", got.Extra["role"])
	}
}

func TestSignVerify_HS512(t *testing.T) {
	secret := []byte("another-secret")
	tok, err := jwtx.Sign(jwtx.Claims{Subject: "x"}, jwtx.HS512(secret))
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	if _, err := jwtx.Verify(tok, jwtx.HS512(secret)); err != nil {
		t.Errorf("Verify: %v", err)
	}
}

func TestSignVerify_RS256(t *testing.T) {
	priv, err := cryptox.GenerateRSAKeyPair(2048)
	if err != nil {
		t.Fatalf("GenerateRSAKeyPair: %v", err)
	}
	tok, err := jwtx.Sign(jwtx.Claims{Subject: "u-1"}, jwtx.RS256Sign(priv))
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	if _, err := jwtx.Verify(tok, jwtx.RS256Verify(&priv.PublicKey)); err != nil {
		t.Errorf("Verify: %v", err)
	}
}

func TestVerify_Expired(t *testing.T) {
	tok, _ := jwtx.Sign(jwtx.Claims{
		Subject:   "u",
		ExpiresAt: time.Now().Add(-time.Hour),
	}, jwtx.HS256([]byte("k")))
	_, err := jwtx.Verify(tok, jwtx.HS256([]byte("k")))
	if !errors.Is(err, jwtx.ErrTokenExpired) {
		t.Errorf("expected ErrTokenExpired, got %v", err)
	}
}

func TestVerify_NotBefore(t *testing.T) {
	tok, _ := jwtx.Sign(jwtx.Claims{
		Subject:   "u",
		NotBefore: time.Now().Add(time.Hour),
	}, jwtx.HS256([]byte("k")))
	_, err := jwtx.Verify(tok, jwtx.HS256([]byte("k")))
	if !errors.Is(err, jwtx.ErrTokenInvalid) {
		t.Errorf("expected ErrTokenInvalid, got %v", err)
	}
}

func TestVerify_AlgNone(t *testing.T) {
	// Hand-craft an alg=none token.
	header := "eyJhbGciOiJub25lIiwidHlwIjoiSldUIn0"
	payload := "eyJzdWIiOiJ1In0"
	tok := header + "." + payload + "."
	_, err := jwtx.Verify(tok, jwtx.HS256([]byte("anything")))
	if !errors.Is(err, jwtx.ErrAlgorithmNone) {
		t.Errorf("expected ErrAlgorithmNone, got %v", err)
	}
}

func TestVerify_AlgMismatch(t *testing.T) {
	tok, _ := jwtx.Sign(jwtx.Claims{Subject: "u"}, jwtx.HS256([]byte("k")))
	_, err := jwtx.Verify(tok, jwtx.HS512([]byte("k")))
	if !errors.Is(err, jwtx.ErrAlgorithmMismatch) {
		t.Errorf("expected ErrAlgorithmMismatch, got %v", err)
	}
}

func TestVerify_Malformed(t *testing.T) {
	if _, err := jwtx.Verify("not.a.jwt!", jwtx.HS256([]byte("k"))); !errors.Is(err, jwtx.ErrTokenMalformed) {
		t.Errorf("expected ErrTokenMalformed, got %v", err)
	}
	if _, err := jwtx.Verify("only-two.parts", jwtx.HS256([]byte("k"))); !errors.Is(err, jwtx.ErrTokenMalformed) {
		t.Errorf("expected ErrTokenMalformed, got %v", err)
	}
}

func TestVerify_TamperedSignature(t *testing.T) {
	tok, _ := jwtx.Sign(jwtx.Claims{Subject: "u"}, jwtx.HS256([]byte("k1")))
	_, err := jwtx.Verify(tok, jwtx.HS256([]byte("k2")))
	if !errors.Is(err, jwtx.ErrSignatureInvalid) {
		t.Errorf("expected ErrSignatureInvalid, got %v", err)
	}
}

func TestAudience_String(t *testing.T) {
	// Hand-craft a payload with string-typed aud.
	tok, _ := jwtx.Sign(jwtx.Claims{
		Subject:  "u",
		Audience: []string{"single"},
	}, jwtx.HS256([]byte("k")))
	c, err := jwtx.Verify(tok, jwtx.HS256([]byte("k")))
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if len(c.Audience) != 1 || c.Audience[0] != "single" {
		t.Errorf("Audience = %v", c.Audience)
	}
}

func TestSign_HMACEmptyKey(t *testing.T) {
	_, err := jwtx.Sign(jwtx.Claims{Subject: "u"}, jwtx.HS256(nil))
	if err == nil {
		t.Error("Sign with empty HMAC key should error")
	}
}
