package jwtx_test

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"errors"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/santekno/sdk/jwtx"
)

// The payload and keys below are RFC 7515 appendices A.2 and A.3. They were
// produced by another implementation, so they pin wire compatibility rather
// than agreement with the helpers in this package.
const (
	rfc7515Payload = "eyJpc3MiOiJqb2UiLA0KICJleHAiOjEzMDA4MTkzODAsDQogImh0dHA6Ly9leGFtcGxlLmNvbS9pc19yb290Ijp0cnVlfQ"

	rfc7515JWKS = `{"keys":[
		{"kty":"EC","crv":"P-256","kid":"a3",
		 "x":"f83OJ3D2xF1Bg8vub9tLe1gHMzV76e8Tus9uPHvRVEU",
		 "y":"x_FEzRu9m36HLN_tue659LNpXW6pCyStikYjKIWI5a0"},
		{"kty":"RSA","kid":"a2","e":"AQAB",
		 "n":"ofgWCuLjybRlzo0tZWJjNiuSfb4p4fAkd_wWJcyQoTbji9k0l8W26mPddxHmfHQp-Vaw-4qPCJrcS2mJPMEzP1Pt0Bm4d4QlL-yRT-SFd2lZS-pCgNMsD1W_YpRPEwOWvG6b32690r2jZ47soMZo9wGzjb_7OMg0LOL-bSf63kpaSHSXndS5z5rexMdbBYUsLA9e-KXBdQOS-UTo7WTBEMa2R2CapHg665xsmtdVMTBQY4uDZlxvb3qCo5ZwKh9kG4LT6_I5IhlJH7aGhyxXFvUK-DWNmoudF8NAco9_h9iaGNj8q2ethFkMLs91kzk2PAcDTW9gb54h4FRWyuXpoQ"}
	]}`

	rfc7515ES256Token = "eyJhbGciOiJFUzI1NiJ9." + rfc7515Payload +
		".DtEhU3ljbEg8L38VWAfUAqOyKAM6-Xx-F4GawxaepmXFCgfTjDxw5djxLa8ISlSApmWQxfKTUJqPP3-Kg6NU1Q"

	rfc7515RS256Token = "eyJhbGciOiJSUzI1NiJ9." + rfc7515Payload +
		".cC4hiUPoj9Eetdgtv3hF80EGrhuB__dzERat0XF9g2VtQgr9PJbu3XOiZj5RZmh7" +
		"AAuHIm4Bh-0Qc_lF5YKt_O8W2Fp5jujGbds9uJdbF9CUAr7t1dnZcAcQjbKBYNX4" +
		"BAynRFdiuB--f_nZLgrnbyTyWzO75vRK5h6xBArLIARNPvkSjtQBMHlb1L07Qe7K" +
		"0GarZRmB_eSN9383LcOLn6_dO--xi12jzDwusC-eOkHWEsqtFZESc6BfI7noOPqv" +
		"hJ1phCnvWh6IeYI2w9QOYEUipUTI8np6LbgGY9Fs98rqVt5AXLIhWkWywlVmtVrB" +
		"p0igcN_IoypGlUPQGe77Rw"
)

func TestVerify_RFC7515Vectors(t *testing.T) {
	defer jwtx.SetNow(func() time.Time { return time.Unix(1300819000, 0) })()

	ks, err := jwtx.ParseKeySet([]byte(rfc7515JWKS))
	if err != nil {
		t.Fatalf("ParseKeySet: %v", err)
	}
	tests := []struct {
		name  string
		kid   string
		token string
		alg   jwtx.Algorithm
	}{
		{"ES256 appendix A.3", "a3", rfc7515ES256Token, jwtx.AlgES256},
		{"RS256 appendix A.2", "a2", rfc7515RS256Token, jwtx.AlgRS256},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, err := ks.Lookup(t.Context(), tt.kid)
			if err != nil {
				t.Fatalf("Lookup: %v", err)
			}
			if key.Alg != tt.alg {
				t.Fatalf("Alg = %s, want %s", key.Alg, tt.alg)
			}
			claims, err := jwtx.Verify(tt.token, key)
			if err != nil {
				t.Fatalf("Verify: %v", err)
			}
			if claims.Issuer != "joe" || claims.Extra["http://example.com/is_root"] != true {
				t.Errorf("claims = %+v", claims)
			}

			tampered := strings.Replace(tt.token, "."+rfc7515Payload+".", "."+b64([]byte(`{"iss":"eve"}`))+".", 1)
			if _, err := jwtx.Verify(tampered, key); !errors.Is(err, jwtx.ErrSignatureInvalid) {
				t.Errorf("tampered payload: got %v", err)
			}
		})
	}
}

func TestSign_ES256Unsupported(t *testing.T) {
	_, err := jwtx.Sign(jwtx.Claims{Subject: "u"}, jwtx.ES256Verify(&ecKey(t).PublicKey))
	if !errors.Is(err, jwtx.ErrUnsupportedAlgo) {
		t.Errorf("got %v", err)
	}
}

func TestVerify_ES256Valid(t *testing.T) {
	priv := ecKey(t)
	tok := signES256(t, priv, es256Header(testKID), validClaims())
	claims, err := jwtx.Verify(tok, jwtx.ES256Verify(&priv.PublicKey))
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if claims.Subject != "user-1" {
		t.Errorf("Subject = %q", claims.Subject)
	}
}

// fixed32 encodes n as a 32-byte big-endian integer.
func fixed32(n *big.Int) []byte { return n.FillBytes(make([]byte, 32)) }

// shortenedSignature returns a signature whose r starts with a zero byte, with
// that byte removed. Some signers get this wrong by emitting a minimal integer.
func shortenedSignature(t *testing.T, priv *ecdsa.PrivateKey, input string) []byte {
	t.Helper()
	for range 100000 {
		sig := rawES256(t, priv, input)
		if sig[0] == 0 {
			return sig[1:]
		}
	}
	t.Fatal("no signature with a leading zero byte found")
	return nil
}

func TestVerify_ES256RejectsBadSignatures(t *testing.T) {
	priv := ecKey(t)
	key := jwtx.ES256Verify(&priv.PublicKey)
	input := signingInput(t, es256Header(testKID), validClaims())
	good := rawES256(t, priv, input)
	r, s := good[:32], good[32:]

	h := sha256.Sum256([]byte(input))
	der, err := ecdsa.SignASN1(rand.Reader, priv, h[:])
	if err != nil {
		t.Fatalf("SignASN1: %v", err)
	}
	order := elliptic.P256().Params().N
	zero := make([]byte, 32)
	join := func(a, b []byte) []byte { return append(append([]byte{}, a...), b...) }
	other, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	flipped := join(r, s)
	flipped[63] ^= 1

	tests := []struct {
		name string
		sig  []byte
	}{
		{"empty", nil},
		{"one byte", []byte{1}},
		{"r only", r},
		{"63 bytes", good[:63]},
		{"65 bytes", append(join(r, s), 0)},
		{"zero byte before s", join(join(r, []byte{0}), s)},
		{"leading zero stripped", shortenedSignature(t, priv, input)},
		{"96 bytes", append(join(r, s), zero...)},
		{"DER", der},
		{"DER cut to 64 bytes", join(der, zero)[:64]},
		{"r zero", join(zero, s)},
		{"s zero", join(r, zero)},
		{"all zero", make([]byte, 64)},
		{"r equals order", join(fixed32(order), s)},
		{"s equals order", join(r, fixed32(order))},
		{"all ones", []byte(strings.Repeat("\xff", 64))},
		{"swapped r and s", join(s, r)},
		{"flipped bit", flipped},
		{"other key", rawES256(t, other, input)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := jwtx.Verify(input+"."+b64(tt.sig), key)
			if !errors.Is(err, jwtx.ErrSignatureInvalid) {
				t.Errorf("got %v, want ErrSignatureInvalid", err)
			}
		})
	}
}

func TestVerify_ES256UnusableKeys(t *testing.T) {
	priv := ecKey(t)
	p384, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	tok := signES256(t, priv, es256Header(testKID), validClaims())

	tests := []struct {
		name string
		pub  *ecdsa.PublicKey
		want error
	}{
		{"nil key", nil, jwtx.ErrTokenInvalid},
		{"no curve", &ecdsa.PublicKey{X: big.NewInt(1), Y: big.NewInt(1)}, jwtx.ErrTokenInvalid},
		{"no coordinates", &ecdsa.PublicKey{Curve: elliptic.P256()}, jwtx.ErrTokenInvalid},
		{"no y", &ecdsa.PublicKey{Curve: elliptic.P256(), X: big.NewInt(1)}, jwtx.ErrTokenInvalid},
		{"P-384 key", &p384.PublicKey, jwtx.ErrTokenInvalid},
		{"point off the curve", &ecdsa.PublicKey{Curve: elliptic.P256(), X: big.NewInt(1), Y: big.NewInt(1)}, jwtx.ErrSignatureInvalid},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := jwtx.Verify(tok, jwtx.ES256Verify(tt.pub))
			if !errors.Is(err, tt.want) {
				t.Errorf("got %v, want %v", err, tt.want)
			}
		})
	}
}

func TestVerify_KeysWithoutMaterial(t *testing.T) {
	tests := []struct {
		name  string
		token string
		key   jwtx.Key
		want  error
	}{
		{"RS256 without a public key", signRS256(t, rsaKey(t), map[string]any{"alg": "RS256"}, validClaims()), jwtx.RS256Verify(nil), jwtx.ErrTokenInvalid},
		{"RS256 with an empty public key", signRS256(t, rsaKey(t), map[string]any{"alg": "RS256"}, validClaims()), jwtx.RS256Verify(&rsa.PublicKey{}), jwtx.ErrSignatureInvalid},
		{"unknown algorithm", signHS256(t, []byte("k"), map[string]any{"alg": "XX"}, validClaims()), jwtx.Key{Alg: "XX"}, jwtx.ErrUnsupportedAlgo},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := jwtx.Verify(tt.token, tt.key); !errors.Is(err, tt.want) {
				t.Errorf("got %v, want %v", err, tt.want)
			}
		})
	}
}

func TestVerify_DoesNotEchoHeaderAlg(t *testing.T) {
	const marker = "SECRET-ALG"
	tok := signHS256(t, []byte("k"), map[string]any{"alg": marker}, validClaims())
	_, err := jwtx.Verify(tok, jwtx.HS256([]byte("k")))
	if !errors.Is(err, jwtx.ErrAlgorithmMismatch) {
		t.Fatalf("got %v", err)
	}
	if strings.Contains(err.Error(), marker) {
		t.Errorf("error echoes the token: %q", err)
	}
}
