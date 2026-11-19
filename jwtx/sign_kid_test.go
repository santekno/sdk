package jwtx_test

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/santekno/sdk/jwtx"
)

type mapKeySet map[string]jwtx.Key

func (m mapKeySet) Lookup(_ context.Context, kid string) (jwtx.Key, error) {
	if k, ok := m[kid]; ok {
		return k, nil
	}
	return jwtx.Key{}, jwtx.ErrKeyNotFound
}

func kidClaims() jwtx.Claims {
	return jwtx.Claims{Subject: "u-1", ExpiresAt: time.Now().Add(time.Hour)}
}

func TestSign_GoldenUnchanged(t *testing.T) {
	c := jwtx.Claims{Subject: "u-42", ExpiresAt: time.Unix(2000000000, 0)}
	// Independent reconstruction of the pre-SignWith output: fixed header and
	// payload bytes, HMAC-SHA256 from the standard library.
	enc := base64.RawURLEncoding.EncodeToString
	input := enc([]byte(`{"alg":"HS256","typ":"JWT"}`)) + "." + enc([]byte(`{"exp":2000000000,"sub":"u-42"}`))
	mac := hmac.New(sha256.New, []byte("secret"))
	mac.Write([]byte(input))
	want := input + "." + enc(mac.Sum(nil))
	for name, fn := range map[string]func() (string, error){
		"Sign":     func() (string, error) { return jwtx.Sign(c, jwtx.HS256([]byte("secret"))) },
		"SignWith": func() (string, error) { return jwtx.SignWith(c, jwtx.HS256([]byte("secret"))) },
	} {
		got, err := fn()
		if err != nil || got != want {
			t.Errorf("%s = %q, %v; want %q", name, got, err, want)
		}
	}
}

func TestSignWith_KeyID(t *testing.T) {
	rsaPriv := rsaKey(t)
	hs := []byte("0123456789abcdef0123456789abcdef")
	tests := []struct {
		name   string
		sign   jwtx.Key
		verify jwtx.Key
		alg    jwtx.Algorithm
	}{
		{"HS256", jwtx.HS256(hs), jwtx.HS256(hs), jwtx.AlgHS256},
		{"HS512", jwtx.HS512(hs), jwtx.HS512(hs), jwtx.AlgHS512},
		{"RS256", jwtx.RS256Sign(rsaPriv), jwtx.Key{Alg: jwtx.AlgRS256, RSAPub: &rsaPriv.PublicKey}, jwtx.AlgRS256},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tok, err := jwtx.SignWith(kidClaims(), tt.sign, jwtx.WithKeyID("k-1.a_B"))
			if err != nil {
				t.Fatal(err)
			}
			ks := mapKeySet{"k-1.a_B": tt.verify}
			got, err := jwtx.VerifyWithKeySet(tok, ks, []jwtx.Algorithm{tt.alg})
			if err != nil || got.Subject != "u-1" {
				t.Fatalf("verify = %v, %v", got, err)
			}
			// unknown kid
			_, err = jwtx.VerifyWithKeySet(tok, mapKeySet{"other": tt.verify}, []jwtx.Algorithm{tt.alg})
			if !errors.Is(err, jwtx.ErrKeyNotFound) {
				t.Errorf("unknown kid err = %v", err)
			}
			// wrong key under the right kid
			bad := mapKeySet{"k-1.a_B": jwtx.HS256([]byte("another-secret-another-secret-12"))}
			if tt.alg != jwtx.AlgHS256 {
				bad["k-1.a_B"] = jwtx.Key{Alg: tt.alg, HMAC: []byte("another-secret-another-secret-12")}
			}
			if _, err = jwtx.VerifyWithKeySet(tok, bad, []jwtx.Algorithm{tt.alg}); err == nil {
				t.Error("wrong key accepted")
			}
		})
	}
}

func TestSignWith_InvalidKeyID(t *testing.T) {
	tests := []struct{ name, kid string }{
		{"empty", ""},
		{"too long", strings.Repeat("a", 65)},
		{"quote", `a"b`},
		{"backslash", `a\b`},
		{"space", "a b"},
		{"newline", "a\nb"},
		{"slash", "a/b"},
		{"non-ascii", "kîd"},
		{"json injection", `x","alg":"none`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := jwtx.SignWith(kidClaims(), jwtx.HS256([]byte("s")), jwtx.WithKeyID(tt.kid))
			if !errors.Is(err, jwtx.ErrTokenInvalid) {
				t.Errorf("err = %v", err)
			}
		})
	}
	if _, err := jwtx.SignWith(kidClaims(), jwtx.HS256([]byte("s")), jwtx.WithKeyID(strings.Repeat("a", 64))); err != nil {
		t.Errorf("64 chars rejected: %v", err)
	}
}
