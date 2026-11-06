package jwtx_test

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"testing"

	"github.com/santekno/sdk/jwtx"
)

// FuzzVerifyWithKeySet feeds arbitrary tokens to the verifier. It must not
// panic, and the only token it may accept is the one properly signed seed.
func FuzzVerifyWithKeySet(f *testing.F) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		f.Fatalf("GenerateKey: %v", err)
	}
	ks, err := jwtx.ParseKeySet(jwksDoc(f, ecJWK(f, &priv.PublicKey, testKID)))
	if err != nil {
		f.Fatalf("ParseKeySet: %v", err)
	}
	claims := map[string]any{"sub": "u", "exp": int64(4102444800)}
	valid := signES256(f, priv, es256Header(testKID), claims)

	f.Add(valid)
	f.Add("")
	f.Add("..")
	f.Add("a.b.c")
	f.Add("....")
	f.Add(rfc7515ES256Token)
	f.Add(signingInput(f, map[string]any{"alg": "none", "kid": testKID}, claims) + ".")
	f.Add(signHS256(f, []byte("secret"), map[string]any{"alg": "HS256", "kid": testKID}, claims))
	f.Add(b64([]byte(`{"alg":"ES256","kid":"key-1","crit":["x"]}`)) + ".e30.AA")

	f.Fuzz(func(t *testing.T, token string) {
		algs := []jwtx.Algorithm{jwtx.AlgES256, jwtx.AlgRS256, jwtx.AlgHS256}
		claims, err := jwtx.VerifyWithKeySet(token, ks, algs)
		if (err == nil) != (claims != nil) {
			t.Fatalf("claims = %v, err = %v", claims, err)
		}
		if err == nil && token != valid {
			t.Fatalf("accepted a token that is not the signed seed: %q", token)
		}
	})
}

// FuzzVerifyES256Signature exercises signature decoding with the rest of the
// token held valid.
func FuzzVerifyES256Signature(f *testing.F) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		f.Fatalf("GenerateKey: %v", err)
	}
	key := jwtx.ES256Verify(&priv.PublicKey)
	input := signingInput(f, es256Header(testKID), map[string]any{"sub": "u"})
	good := rawES256(f, priv, input)

	f.Add(good)
	f.Add([]byte{})
	f.Add(make([]byte, 64))
	f.Add(make([]byte, 65))

	f.Fuzz(func(t *testing.T, sig []byte) {
		claims, err := jwtx.Verify(input+"."+b64(sig), key)
		if err == nil && claims == nil {
			t.Fatal("nil claims without an error")
		}
		if err == nil && len(sig) != 64 {
			t.Fatalf("accepted a %d-byte signature", len(sig))
		}
	})
}

// FuzzParseKeySet feeds arbitrary documents to the parser. It must not panic,
// and every key it returns must be usable without panicking.
func FuzzParseKeySet(f *testing.F) {
	ec, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		f.Fatalf("GenerateKey: %v", err)
	}
	f.Add(jwksDoc(f, ecJWK(f, &ec.PublicKey, testKID)))
	f.Add([]byte(rfc7515JWKS))
	f.Add([]byte(``))
	f.Add([]byte(`{}`))
	f.Add([]byte(`null`))
	f.Add([]byte(`{"keys":null}`))
	f.Add([]byte(`{"keys":[null,1,"x",[],{}]}`))
	f.Add([]byte(`{"keys":[{"kty":"EC","crv":"P-256","kid":"a","x":"","y":""}]}`))
	f.Add([]byte(`{"keys":[{"kty":"RSA","kid":"a","n":"AA","e":"AA"}]}`))
	f.Add([]byte(`{"keys":[{"kty":"oct","kid":"a","k":"AA"}]}`))

	token := signES256(f, ec, es256Header("a"), map[string]any{"exp": int64(4102444800)})
	f.Fuzz(func(t *testing.T, data []byte) {
		ks, err := jwtx.ParseKeySet(data)
		if err != nil {
			if ks != nil {
				t.Fatal("key set returned with an error")
			}
			return
		}
		if ks.Len() == 0 {
			t.Fatal("empty key set without an error")
		}
		// The outcome depends on the document; only a panic is a failure.
		_, _ = jwtx.VerifyWithKeySet(token, ks, []jwtx.Algorithm{jwtx.AlgES256})
	})
}
