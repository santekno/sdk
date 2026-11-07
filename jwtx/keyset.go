package jwtx

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"math/big"
	"slices"
)

const (
	// maxJWKSBytes bounds a JWKS document, fetched or parsed.
	maxJWKSBytes = 64 << 10
	// p256CoordinateSize is the exact width of "x" and "y" (RFC 7518 section
	// 6.2.1.2).
	p256CoordinateSize = 32
	// minRSABits follows RFC 7518 section 3.3. maxRSABits bounds the cost of
	// one verification, which an unauthenticated token can trigger.
	minRSABits = 2048
	maxRSABits = 8192
	// maxRSAExponent is the largest public exponent crypto/rsa accepts.
	maxRSAExponent = 1<<31 - 1
)

// KeySet resolves the verification key named by a token's "kid".
// StaticKeySet and JWKSClient implement it.
type KeySet interface {
	// Lookup returns the key for kid. It returns ErrKeyNotFound when kid is
	// unknown and ErrKeySetUnavailable when no key can be served at all.
	Lookup(ctx context.Context, kid string) (Key, error)
}

// StaticKeySet is an immutable set of public keys parsed from a JWKS document.
type StaticKeySet struct {
	keys map[string]Key
}

// Len returns the number of usable keys.
func (s *StaticKeySet) Len() int {
	if s == nil {
		return 0
	}
	return len(s.keys)
}

// Lookup implements KeySet.
func (s *StaticKeySet) Lookup(_ context.Context, kid string) (Key, error) {
	if key, ok := s.get(kid); ok {
		return key, nil
	}
	return Key{}, ErrKeyNotFound
}

func (s *StaticKeySet) get(kid string) (Key, bool) {
	if s == nil {
		return Key{}, false
	}
	key, ok := s.keys[kid]
	return key, ok
}

// jwk holds the members of a JSON Web Key that jwtx reads. Private members
// such as "d" are deliberately absent, as are symmetric "oct" keys: a key set
// only ever yields public keys.
type jwk struct {
	Kty    string   `json:"kty"`
	Kid    string   `json:"kid"`
	Use    string   `json:"use"`
	KeyOps []string `json:"key_ops"`
	Alg    string   `json:"alg"`
	Crv    string   `json:"crv"`
	X      string   `json:"x"`
	Y      string   `json:"y"`
	N      string   `json:"n"`
	E      string   `json:"e"`
}

// ParseKeySet parses a JWKS document (RFC 7517). It keeps EC P-256 keys for
// ES256 and RSA keys of 2048 to 8192 bits for RS256. Any other entry is
// skipped: a missing "kid", a "use" or "key_ops" that excludes signature
// verification, an unsupported "alg" or key type, or invalid key material such
// as a point outside the curve. A key without "alg" takes it from its type.
// When a "kid" repeats, the first usable entry wins.
//
// It returns ErrJWKSInvalid when data is not a JWKS, exceeds 64 KiB, or holds
// no usable key.
func ParseKeySet(data []byte) (*StaticKeySet, error) {
	if len(data) > maxJWKSBytes {
		return nil, fmt.Errorf("%w: document too large", ErrJWKSInvalid)
	}
	var doc struct {
		Keys []json.RawMessage `json:"keys"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("%w: not a JWKS document", ErrJWKSInvalid)
	}

	keys := make(map[string]Key, len(doc.Keys))
	for _, entry := range doc.Keys {
		var j jwk
		if err := json.Unmarshal(entry, &j); err != nil {
			continue
		}
		key, ok := j.publicKey()
		if !ok {
			continue
		}
		if _, dup := keys[j.Kid]; !dup {
			keys[j.Kid] = key
		}
	}
	if len(keys) == 0 {
		return nil, fmt.Errorf("%w: no usable key", ErrJWKSInvalid)
	}
	return &StaticKeySet{keys: keys}, nil
}

func (j jwk) publicKey() (Key, bool) {
	if j.Kid == "" || !j.forVerification() {
		return Key{}, false
	}
	switch j.Kty {
	case "EC":
		return j.ecKey()
	case "RSA":
		return j.rsaKey()
	default:
		return Key{}, false
	}
}

func (j jwk) forVerification() bool {
	return (j.Use == "" || j.Use == "sig") &&
		(len(j.KeyOps) == 0 || slices.Contains(j.KeyOps, "verify"))
}

func (j jwk) ecKey() (Key, bool) {
	if (j.Alg != "" && j.Alg != string(AlgES256)) || j.Crv != "P-256" {
		return Key{}, false
	}
	x, err := strictB64.DecodeString(j.X)
	if err != nil || len(x) != p256CoordinateSize {
		return Key{}, false
	}
	y, err := strictB64.DecodeString(j.Y)
	if err != nil || len(y) != p256CoordinateSize {
		return Key{}, false
	}
	// The parser rejects points that are off the curve or at infinity.
	pub, err := ecdsa.ParseUncompressedPublicKey(elliptic.P256(), slices.Concat([]byte{4}, x, y))
	if err != nil {
		return Key{}, false
	}
	return ES256Verify(pub), true
}

func (j jwk) rsaKey() (Key, bool) {
	if j.Alg != "" && j.Alg != string(AlgRS256) {
		return Key{}, false
	}
	nBytes, err := strictB64.DecodeString(j.N)
	if err != nil {
		return Key{}, false
	}
	eBytes, err := strictB64.DecodeString(j.E)
	if err != nil {
		return Key{}, false
	}
	n := new(big.Int).SetBytes(nBytes)
	e := new(big.Int).SetBytes(eBytes)
	if n.BitLen() < minRSABits || n.BitLen() > maxRSABits || n.Bit(0) == 0 {
		return Key{}, false
	}
	if e.Bit(0) == 0 || e.Cmp(big.NewInt(3)) < 0 || e.Cmp(big.NewInt(maxRSAExponent)) > 0 {
		return Key{}, false
	}
	return RS256Verify(&rsa.PublicKey{N: n, E: int(e.Int64())}), true
}
