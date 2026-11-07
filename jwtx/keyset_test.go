package jwtx_test

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"errors"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/santekno/sdk/jwtx"
)

var allowAsymmetric = []jwtx.Algorithm{jwtx.AlgES256, jwtx.AlgRS256}

func parseSet(t *testing.T, doc []byte) *jwtx.StaticKeySet {
	t.Helper()
	ks, err := jwtx.ParseKeySet(doc)
	if err != nil {
		t.Fatalf("ParseKeySet: %v", err)
	}
	return ks
}

// A document shaped like the JWKS endpoint of Supabase Auth.
func TestParseKeySet_SupabaseShape(t *testing.T) {
	jwk := ecJWK(t, &ecKey(t).PublicKey, "b1f7a0b2")
	jwk["ext"] = true
	jwk["key_ops"] = []string{"verify"}
	ks := parseSet(t, jwksDoc(t, jwk))

	tok := signES256(t, ecKey(t), es256Header("b1f7a0b2"), validClaims())
	claims, err := jwtx.VerifyWithKeySet(tok, ks, allowAsymmetric)
	if err != nil {
		t.Fatalf("VerifyWithKeySet: %v", err)
	}
	meta, ok := claims.Extra["app_metadata"].(map[string]any)
	if !ok || meta["role"] != "editor" {
		t.Errorf("app_metadata = %v", claims.Extra["app_metadata"])
	}
}

// oddModulus returns a big-endian integer of top followed by size bytes, with
// the lowest bit set so that only its length can disqualify it.
func oddModulus(top byte, size int) []byte {
	n := append([]byte{top}, make([]byte, size)...)
	n[len(n)-1] = 1
	return n
}

func TestParseKeySet_SkipsUnusableKeys(t *testing.T) {
	ecPub := &ecKey(t).PublicKey
	rsaPub := &rsaKey(t).PublicKey
	p384, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	p384Point, err := p384.PublicKey.Bytes()
	if err != nil {
		t.Fatalf("Bytes: %v", err)
	}
	ecWith := func(k string, v any) map[string]any {
		j := ecJWK(t, ecPub, "bad")
		j[k] = v
		return j
	}
	ecWithout := func(k string) map[string]any {
		j := ecJWK(t, ecPub, "bad")
		delete(j, k)
		return j
	}
	rsaWith := func(k string, v any) map[string]any {
		j := rsaJWK(rsaPub, "bad")
		j[k] = v
		return j
	}
	smallRSA, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	point, err := ecPub.Bytes()
	if err != nil {
		t.Fatalf("Bytes: %v", err)
	}
	movedByte := ecWith("x", b64(point[1:32]))
	movedByte["y"] = b64(point[32:])
	offCurveX := ecJWK(t, ecPub, "bad")
	offCurveX["x"] = b64(bytes.Repeat([]byte{7}, 32))
	p := elliptic.P256().Params().P

	tests := []struct {
		name string
		key  any
	}{
		{"no kid", ecWithout("kid")},
		{"encryption key", ecWith("use", "enc")},
		{"key_ops without verify", ecWith("key_ops", []string{"encrypt"})},
		{"unsupported alg", ecWith("alg", "ES384")},
		{"EC key marked RS256", ecWith("alg", "RS256")},
		{"lower-case alg", ecWith("alg", "es256")},
		{"EC key marked HS256", ecWith("alg", "HS256")},
		{"P-384 curve", ecWith("crv", "P-384")},
		{"P-384 point labelled P-256", ecWith("x", b64(p384Point[1:49]))},
		{"point off the curve", offCurveX},
		{"point at origin", map[string]any{"kty": "EC", "crv": "P-256", "kid": "bad", "x": b64(make([]byte, 32)), "y": b64(make([]byte, 32))}},
		{"coordinate equal to the field prime", ecWith("x", b64(p.Bytes()))},
		{"short x", ecWith("x", b64(make([]byte, 31)))},
		{"long y", ecWith("y", b64(make([]byte, 33)))},
		{"byte moved from x to y", movedByte},
		{"x not base64url", ecWith("x", "%%%")},
		{"y not base64url", ecWith("y", "%%%")},
		{"x of wrong JSON type", ecWith("x", 12)},
		{"unknown kty", map[string]any{"kty": "OKP", "kid": "bad"}},
		{"symmetric key", map[string]any{"kty": "oct", "kid": "bad", "k": "c2VjcmV0", "alg": "HS256"}},
		{"RSA key marked ES256", rsaWith("alg", "ES256")},
		{"RSA PS256", rsaWith("alg", "PS256")},
		{"RSA 1024 bits", rsaJWK(&smallRSA.PublicKey, "bad")},
		{"RSA 2047 bits", rsaWith("n", b64(new(big.Int).Rsh(rsaPub.N, 1).Bytes()))},
		{"RSA 8193 bits", rsaWith("n", b64(oddModulus(1, 1024)))},
		{"RSA even modulus", rsaWith("n", b64(new(big.Int).Sub(rsaPub.N, big.NewInt(1)).Bytes()))},
		{"RSA modulus not base64url", rsaWith("n", "%%%")},
		{"RSA exponent not base64url", rsaWith("e", "%%%")},
		{"RSA exponent 1", rsaWith("e", "AQ")},
		{"RSA even exponent", rsaWith("e", "AQAA")},
		{"RSA exponent zero", rsaWith("e", "")},
		{"RSA exponent over 31 bits", rsaWith("e", "AQAAAAE")},
		{"entry that is not an object", 42},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := json.Marshal(map[string]any{"keys": []any{tt.key, ecJWK(t, ecPub, "good")}})
			if err != nil {
				t.Fatalf("Marshal: %v", err)
			}
			ks := parseSet(t, doc)
			if ks.Len() != 1 {
				t.Fatalf("Len = %d, want only the good key", ks.Len())
			}
			if _, err := ks.Lookup(t.Context(), "bad"); !errors.Is(err, jwtx.ErrKeyNotFound) {
				t.Errorf("Lookup(bad): got %v", err)
			}
			if _, err := ks.Lookup(t.Context(), "good"); err != nil {
				t.Errorf("Lookup(good): %v", err)
			}
		})
	}
}

func TestParseKeySet_AcceptsKeysWithoutOptionalMembers(t *testing.T) {
	ec := ecJWK(t, &ecKey(t).PublicKey, "ec")
	delete(ec, "alg")
	delete(ec, "use")
	rs := rsaJWK(&rsaKey(t).PublicKey, "rs")
	delete(rs, "alg")
	delete(rs, "use")
	ks := parseSet(t, jwksDoc(t, ec, rs))

	tests := []struct {
		kid  string
		want jwtx.Algorithm
	}{{"ec", jwtx.AlgES256}, {"rs", jwtx.AlgRS256}}
	for _, tt := range tests {
		key, err := ks.Lookup(t.Context(), tt.kid)
		if err != nil {
			t.Fatalf("Lookup(%s): %v", tt.kid, err)
		}
		if key.Alg != tt.want {
			t.Errorf("Alg = %s, want %s", key.Alg, tt.want)
		}
	}
}

func TestParseKeySet_AcceptsLargestRSAModulus(t *testing.T) {
	j := rsaJWK(&rsaKey(t).PublicKey, testKID)
	j["n"] = b64(oddModulus(0x80, 1023))
	if ks := parseSet(t, jwksDoc(t, j)); ks.Len() != 1 {
		t.Errorf("Len = %d", ks.Len())
	}
}

func TestParseKeySet_DuplicateKidKeepsFirst(t *testing.T) {
	other, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	ks := parseSet(t, jwksDoc(t, ecJWK(t, &ecKey(t).PublicKey, testKID), ecJWK(t, &other.PublicKey, testKID)))

	if _, err := jwtx.VerifyWithKeySet(signES256(t, ecKey(t), es256Header(testKID), validClaims()), ks, allowAsymmetric); err != nil {
		t.Errorf("first key: %v", err)
	}
	if _, err := jwtx.VerifyWithKeySet(signES256(t, other, es256Header(testKID), validClaims()), ks, allowAsymmetric); !errors.Is(err, jwtx.ErrSignatureInvalid) {
		t.Errorf("second key: got %v", err)
	}
}

func TestParseKeySet_RejectsDocuments(t *testing.T) {
	good := jwksDoc(t, ecJWK(t, &ecKey(t).PublicKey, testKID))
	oversized := append(good, bytes.Repeat([]byte(" "), 64<<10+1-len(good))...)
	tests := []struct {
		name string
		doc  []byte
	}{
		{"nil", nil},
		{"empty", []byte{}},
		{"not JSON", []byte("<html>")},
		{"array", []byte("[]")},
		{"null", []byte("null")},
		{"no keys member", []byte(`{}`)},
		{"keys of wrong type", []byte(`{"keys":"x"}`)},
		{"empty keys", []byte(`{"keys":[]}`)},
		{"only unusable keys", []byte(`{"keys":[{"kty":"oct","kid":"a","k":"AA"}]}`)},
		{"valid document one byte over the limit", oversized},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := jwtx.ParseKeySet(tt.doc); !errors.Is(err, jwtx.ErrJWKSInvalid) {
				t.Errorf("got %v", err)
			}
		})
	}
}

func TestParseKeySet_AcceptsDocumentAtTheLimit(t *testing.T) {
	doc := jwksDoc(t, ecJWK(t, &ecKey(t).PublicKey, testKID))
	padded := append(doc, bytes.Repeat([]byte(" "), 64<<10-len(doc))...)
	if _, err := jwtx.ParseKeySet(padded); err != nil {
		t.Errorf("ParseKeySet: %v", err)
	}
}

func TestStaticKeySet_NilIsEmpty(t *testing.T) {
	var ks *jwtx.StaticKeySet
	if ks.Len() != 0 {
		t.Errorf("Len = %d", ks.Len())
	}
	if _, err := ks.Lookup(context.Background(), "x"); !errors.Is(err, jwtx.ErrKeyNotFound) {
		t.Errorf("got %v", err)
	}
}

func TestVerifyWithKeySet_ValidTokens(t *testing.T) {
	ks := parseSet(t, jwksDoc(t, ecJWK(t, &ecKey(t).PublicKey, "ec"), rsaJWK(&rsaKey(t).PublicKey, "rs")))
	tests := []struct {
		name  string
		token string
	}{
		{"ES256", signES256(t, ecKey(t), es256Header("ec"), validClaims())},
		{"RS256", signRS256(t, rsaKey(t), map[string]any{"alg": "RS256", "kid": "rs"}, validClaims())},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims, err := jwtx.VerifyWithKeySet(tt.token, ks, allowAsymmetric)
			if err != nil {
				t.Fatalf("VerifyWithKeySet: %v", err)
			}
			if claims.Subject != "user-1" || claims.Issuer != "https://issuer.test/auth/v1" {
				t.Errorf("claims = %+v", claims)
			}
			if len(claims.Audience) != 1 || claims.Audience[0] != "authenticated" {
				t.Errorf("Audience = %v", claims.Audience)
			}
		})
	}
}

func TestVerifyWithKeySet_AlgorithmAttacks(t *testing.T) {
	ecPriv, rsaPriv := ecKey(t), rsaKey(t)
	ks := parseSet(t, jwksDoc(t, ecJWK(t, &ecPriv.PublicKey, "ec"), rsaJWK(&rsaPriv.PublicKey, "rs")))
	claims := validClaims()

	pubDER, err := x509.MarshalPKIXPublicKey(&rsaPriv.PublicKey)
	if err != nil {
		t.Fatalf("MarshalPKIXPublicKey: %v", err)
	}
	ecPoint, err := ecPriv.PublicKey.Bytes()
	if err != nil {
		t.Fatalf("Bytes: %v", err)
	}
	withAlg := func(alg any, kid string) map[string]any { return map[string]any{"alg": alg, "kid": kid} }
	everyAlg := []jwtx.Algorithm{jwtx.AlgES256, jwtx.AlgRS256, jwtx.AlgHS256, jwtx.AlgHS512, "none", "None", ""}

	unsigned := func(header map[string]any) string { return signingInput(t, header, claims) + "." }
	tests := []struct {
		name    string
		token   string
		allowed []jwtx.Algorithm
		want    error
	}{
		{"none", unsigned(withAlg("none", "ec")), allowAsymmetric, jwtx.ErrAlgorithmNone},
		{"None", unsigned(withAlg("None", "ec")), allowAsymmetric, jwtx.ErrAlgorithmNone},
		{"NONE", unsigned(withAlg("NONE", "ec")), allowAsymmetric, jwtx.ErrAlgorithmNone},
		{"none despite being allowed", unsigned(withAlg("none", "ec")), everyAlg, jwtx.ErrAlgorithmNone},
		{"empty alg", unsigned(withAlg("", "ec")), everyAlg, jwtx.ErrAlgorithmNone},
		{"null alg", unsigned(withAlg(nil, "ec")), everyAlg, jwtx.ErrAlgorithmNone},
		{"missing alg", unsigned(map[string]any{"kid": "ec"}), everyAlg, jwtx.ErrAlgorithmNone},
		{"upper-case member name", unsigned(map[string]any{"ALG": "ES256", "kid": "ec"}), everyAlg, jwtx.ErrAlgorithmNone},
		{"alg of wrong JSON type", unsigned(withAlg(7, "ec")), everyAlg, jwtx.ErrTokenMalformed},
		{"lower-case es256", signES256(t, ecPriv, withAlg("es256", "ec"), claims), allowAsymmetric, jwtx.ErrAlgorithmNotAllowed},
		{"mixed-case Es256", signES256(t, ecPriv, withAlg("Es256", "ec"), claims), allowAsymmetric, jwtx.ErrAlgorithmNotAllowed},
		{"lower-case rs256", signRS256(t, rsaPriv, withAlg("rs256", "rs"), claims), allowAsymmetric, jwtx.ErrAlgorithmNotAllowed},
		{"alg outside the list", signES256(t, ecPriv, es256Header("ec"), claims), []jwtx.Algorithm{jwtx.AlgRS256}, jwtx.ErrAlgorithmNotAllowed},
		{"empty allow-list", signES256(t, ecPriv, es256Header("ec"), claims), nil, jwtx.ErrAlgorithmNotAllowed},
		{"HS256 by default", signHS256(t, []byte("x"), withAlg("HS256", "rs"), claims), allowAsymmetric, jwtx.ErrAlgorithmNotAllowed},
		{"HS256 with RSA modulus as secret", signHS256(t, rsaPriv.N.Bytes(), withAlg("HS256", "rs"), claims), everyAlg, jwtx.ErrAlgorithmMismatch},
		{"HS256 with DER public key as secret", signHS256(t, pubDER, withAlg("HS256", "rs"), claims), everyAlg, jwtx.ErrAlgorithmMismatch},
		{"HS256 with EC point as secret", signHS256(t, ecPoint, withAlg("HS256", "ec"), claims), everyAlg, jwtx.ErrAlgorithmMismatch},
		{"HS512 against RSA key", signHS256(t, []byte("x"), withAlg("HS512", "rs"), claims), everyAlg, jwtx.ErrAlgorithmMismatch},
		{"RS256 alg on the EC key", signRS256(t, rsaPriv, withAlg("RS256", "ec"), claims), allowAsymmetric, jwtx.ErrAlgorithmMismatch},
		{"ES256 alg on the RSA key", signES256(t, ecPriv, withAlg("ES256", "rs"), claims), allowAsymmetric, jwtx.ErrAlgorithmMismatch},
		{"RS256 bytes under an ES256 header", swapSignature(t, signES256(t, ecPriv, es256Header("ec"), claims), signRS256(t, rsaPriv, withAlg("RS256", "rs"), claims)), allowAsymmetric, jwtx.ErrSignatureInvalid},
		{"crit header", signES256(t, ecPriv, map[string]any{"alg": "ES256", "kid": "ec", "crit": []string{"exp"}, "exp": 1}, claims), allowAsymmetric, jwtx.ErrTokenInvalid},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := jwtx.VerifyWithKeySet(tt.token, ks, tt.allowed)
			if !errors.Is(err, tt.want) {
				t.Fatalf("got %v, want %v", err, tt.want)
			}
			if got != nil {
				t.Errorf("claims returned with an error: %+v", got)
			}
		})
	}
}

// swapSignature returns token with the signature segment of donor.
func swapSignature(t *testing.T, token, donor string) string {
	t.Helper()
	return token[:strings.LastIndexByte(token, '.')] + donor[strings.LastIndexByte(donor, '.'):]
}

// A token must not be able to bring its own key: a verifier that honored
// these headers would accept a token signed by anyone.
func TestVerifyWithKeySet_IgnoresKeysEmbeddedInTheHeader(t *testing.T) {
	attacker, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	ks := parseSet(t, jwksDoc(t, ecJWK(t, &ecKey(t).PublicKey, "ec")))
	header := es256Header("ec")
	header["jwk"] = ecJWK(t, &attacker.PublicKey, "ec")
	header["jku"] = "https://attacker.test/jwks.json"

	if _, err := jwtx.VerifyWithKeySet(signES256(t, attacker, header, validClaims()), ks, allowAsymmetric); !errors.Is(err, jwtx.ErrSignatureInvalid) {
		t.Errorf("got %v", err)
	}
}

func TestVerifyWithKeySet_KeyID(t *testing.T) {
	priv := ecKey(t)
	ks := parseSet(t, jwksDoc(t, ecJWK(t, &priv.PublicKey, testKID)))
	claims := validClaims()
	tests := []struct {
		name   string
		header map[string]any
		want   error
	}{
		{"missing", map[string]any{"alg": "ES256"}, jwtx.ErrKeyIDMissing},
		{"empty", map[string]any{"alg": "ES256", "kid": ""}, jwtx.ErrKeyIDMissing},
		{"null", map[string]any{"alg": "ES256", "kid": nil}, jwtx.ErrKeyIDMissing},
		{"upper-case member name", map[string]any{"alg": "ES256", "KID": testKID}, jwtx.ErrKeyIDMissing},
		{"wrong JSON type", map[string]any{"alg": "ES256", "kid": 1}, jwtx.ErrTokenMalformed},
		{"unknown", map[string]any{"alg": "ES256", "kid": "other"}, jwtx.ErrKeyNotFound},
		{"differs by case", map[string]any{"alg": "ES256", "kid": strings.ToUpper(testKID)}, jwtx.ErrKeyNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := jwtx.VerifyWithKeySet(signES256(t, priv, tt.header, claims), ks, allowAsymmetric)
			if !errors.Is(err, tt.want) {
				t.Errorf("got %v, want %v", err, tt.want)
			}
		})
	}
}

func TestVerifyWithKeySet_NilKeySet(t *testing.T) {
	_, err := jwtx.VerifyWithKeySet("a.b.c", nil, allowAsymmetric)
	if !errors.Is(err, jwtx.ErrKeySetUnavailable) {
		t.Errorf("got %v", err)
	}
}

func TestVerifyWithKeySet_MalformedTokens(t *testing.T) {
	priv := ecKey(t)
	ks := parseSet(t, jwksDoc(t, ecJWK(t, &priv.PublicKey, testKID)))
	good := signES256(t, priv, es256Header(testKID), validClaims())
	parts := strings.Split(good, ".")
	header := b64JSON(t, es256Header(testKID))

	signed := func(payload string) string {
		input := header + "." + b64([]byte(payload))
		return input + "." + b64(rawES256(t, priv, input))
	}
	tests := []struct {
		name  string
		token string
		want  error
	}{
		{"empty", "", jwtx.ErrTokenMalformed},
		{"one segment", "abc", jwtx.ErrTokenMalformed},
		{"two segments", "abc.def", jwtx.ErrTokenMalformed},
		{"four segments", good + ".x", jwtx.ErrTokenMalformed},
		{"only dots", strings.Repeat(".", 1000), jwtx.ErrTokenMalformed},
		{"over the size limit", good + strings.Repeat("A", 64<<10), jwtx.ErrTokenMalformed},
		{"line break in the signature", good + "\n", jwtx.ErrTokenMalformed},
		{"carriage return in the header", "\r" + good, jwtx.ErrTokenMalformed},
		{"surrounding space", " " + good, jwtx.ErrTokenMalformed},
		{"header not base64url", "%%%." + parts[1] + "." + parts[2], jwtx.ErrTokenMalformed},
		{"standard base64 padding", parts[0] + "=." + parts[1] + "." + parts[2], jwtx.ErrTokenMalformed},
		{"header not JSON", b64([]byte("nope")) + "." + parts[1] + "." + parts[2], jwtx.ErrTokenMalformed},
		{"header is an array", b64([]byte("[]")) + "." + parts[1] + "." + parts[2], jwtx.ErrTokenMalformed},
		{"header is null", b64([]byte("null")) + "." + parts[1] + "." + parts[2], jwtx.ErrAlgorithmNone},
		{"signature not base64url", parts[0] + "." + parts[1] + ".%%%", jwtx.ErrTokenMalformed},
		{"signature with non-zero trailing bits", parts[0] + "." + parts[1] + "." + parts[2][:len(parts[2])-1] + "B", jwtx.ErrTokenMalformed},
		{"empty signature", parts[0] + "." + parts[1] + ".", jwtx.ErrSignatureInvalid},
		{"payload not base64url", func() string {
			input := header + ".%%%"
			return input + "." + b64(rawES256(t, priv, input))
		}(), jwtx.ErrTokenMalformed},
		{"payload not JSON", signed("nope"), jwtx.ErrTokenMalformed},
		{"payload is an array", signed("[]"), jwtx.ErrTokenMalformed},
		{"payload is null", signed("null"), jwtx.ErrTokenInvalid},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := jwtx.VerifyWithKeySet(tt.token, ks, allowAsymmetric); !errors.Is(err, tt.want) {
				t.Errorf("got %v, want %v", err, tt.want)
			}
		})
	}
}

func TestVerifyWithKeySet_TimeClaims(t *testing.T) {
	priv := ecKey(t)
	ks := parseSet(t, jwksDoc(t, ecJWK(t, &priv.PublicKey, testKID)))
	now := time.Now()
	future := now.Add(time.Hour).Unix()
	secondsAgo := func(s int) int64 { return now.Add(-time.Duration(s) * time.Second).Unix() }
	secondsAhead := func(s int) int64 { return now.Add(time.Duration(s) * time.Second).Unix() }

	tests := []struct {
		name   string
		claims map[string]any
		leeway time.Duration
		want   error
	}{
		{"valid without leeway", map[string]any{"exp": future}, 0, nil},
		{"exp is required", map[string]any{"sub": "u"}, 0, jwtx.ErrTokenInvalid},
		{"exp null", map[string]any{"exp": nil}, 0, jwtx.ErrTokenMalformed},
		{"exp string", map[string]any{"exp": "2099"}, 0, jwtx.ErrTokenMalformed},
		{"exp zero", map[string]any{"exp": 0}, 0, jwtx.ErrTokenExpired},
		{"exp negative", map[string]any{"exp": -1}, 0, jwtx.ErrTokenMalformed},
		{"exp past the representable range", map[string]any{"exp": 1e300}, 0, jwtx.ErrTokenMalformed},
		{"exp one second over the range", map[string]any{"exp": 253402300800.0}, 0, jwtx.ErrTokenMalformed},
		{"exp at the end of the range", map[string]any{"exp": 253402300799.0}, 0, nil},
		{"expired", map[string]any{"exp": secondsAgo(10)}, 0, jwtx.ErrTokenExpired},
		{"expired within leeway", map[string]any{"exp": secondsAgo(10)}, 30 * time.Second, nil},
		{"expired beyond leeway", map[string]any{"exp": secondsAgo(40)}, 30 * time.Second, jwtx.ErrTokenExpired},
		{"negative leeway counts as zero", map[string]any{"exp": secondsAhead(10)}, -time.Hour, nil},
		{"nbf in the past", map[string]any{"exp": future, "nbf": secondsAgo(10)}, 0, nil},
		{"nbf in the future", map[string]any{"exp": future, "nbf": secondsAhead(60)}, 0, jwtx.ErrTokenInvalid},
		{"nbf within leeway", map[string]any{"exp": future, "nbf": secondsAhead(10)}, 30 * time.Second, nil},
		{"nbf beyond leeway", map[string]any{"exp": future, "nbf": secondsAhead(60)}, 30 * time.Second, jwtx.ErrTokenInvalid},
		{"nbf string", map[string]any{"exp": future, "nbf": "later"}, 0, jwtx.ErrTokenMalformed},
		{"nbf null", map[string]any{"exp": future, "nbf": nil}, 0, jwtx.ErrTokenMalformed},
		{"iat in the future", map[string]any{"exp": future, "iat": secondsAhead(60)}, 0, jwtx.ErrTokenInvalid},
		{"iat within leeway", map[string]any{"exp": future, "iat": secondsAhead(10)}, 30 * time.Second, nil},
		{"iat beyond leeway", map[string]any{"exp": future, "iat": secondsAhead(60)}, 30 * time.Second, jwtx.ErrTokenInvalid},
		{"iat boolean", map[string]any{"exp": future, "iat": true}, 0, jwtx.ErrTokenMalformed},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tok := signES256(t, priv, es256Header(testKID), tt.claims)
			_, err := jwtx.VerifyWithKeySet(tok, ks, allowAsymmetric, jwtx.WithLeeway(tt.leeway))
			if !errors.Is(err, tt.want) {
				t.Errorf("got %v, want %v", err, tt.want)
			}
		})
	}
}

// A token is expired at the instant of "exp", not one tick after it.
func TestVerifyWithKeySet_ExpiredAtExactInstant(t *testing.T) {
	priv := ecKey(t)
	ks := parseSet(t, jwksDoc(t, ecJWK(t, &priv.PublicKey, testKID)))
	instant := time.Unix(2000000000, 0)
	defer jwtx.SetNow(func() time.Time { return instant })()

	tests := []struct {
		name string
		exp  int64
		want error
	}{
		{"one second left", instant.Unix() + 1, nil},
		{"at exp", instant.Unix(), jwtx.ErrTokenExpired},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tok := signES256(t, priv, es256Header(testKID), map[string]any{"exp": tt.exp})
			if _, err := jwtx.VerifyWithKeySet(tok, ks, allowAsymmetric); !errors.Is(err, tt.want) {
				t.Errorf("got %v, want %v", err, tt.want)
			}
		})
	}
}

func TestVerifyWithKeySet_ErrorsDoNotEchoInput(t *testing.T) {
	const marker = "SECRET-MARKER"
	priv := ecKey(t)
	ks := parseSet(t, jwksDoc(t, ecJWK(t, &priv.PublicKey, testKID)))
	claims := validClaims()
	claims["sub"] = marker

	expired := validClaims()
	expired["sub"] = marker
	expired["exp"] = 1
	valid := signES256(t, priv, es256Header(testKID), claims)
	tests := []struct {
		name  string
		token string
	}{
		{"alg", signES256(t, priv, map[string]any{"alg": marker, "kid": testKID}, claims)},
		{"kid", signES256(t, priv, es256Header(marker), claims)},
		{"claims with a bad signature", valid[:strings.LastIndexByte(valid, '.')] + "." + b64([]byte(marker))},
		{"expired claims", signES256(t, priv, es256Header(testKID), expired)},
		{"malformed token", marker},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := jwtx.VerifyWithKeySet(tt.token, ks, allowAsymmetric)
			if err == nil {
				t.Fatal("token accepted")
			}
			if strings.Contains(err.Error(), marker) || strings.Contains(err.Error(), tt.token) {
				t.Errorf("error echoes the token: %q", err)
			}
		})
	}
}

// ParseKeySet errors must not quote the document, which encoding/json errors do.
func TestParseKeySet_ErrorsDoNotEchoInput(t *testing.T) {
	const marker = "SECRET-MARKER"
	tests := []string{
		marker,
		`{"keys":"` + marker + `"}`,
		`{"keys":[{"kty":"oct","kid":"a","k":"` + marker + `"}]}`,
	}
	for _, doc := range tests {
		_, err := jwtx.ParseKeySet([]byte(doc))
		if err == nil {
			t.Fatalf("accepted %q", doc)
		}
		if strings.Contains(err.Error(), marker) || strings.Contains(err.Error(), "SECRET") {
			t.Errorf("error echoes the document: %q", err)
		}
	}
}
