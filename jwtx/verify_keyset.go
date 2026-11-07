package jwtx

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"slices"
	"strings"
	"time"
)

const (
	// maxTokenBytes bounds the work done on an unauthenticated token.
	maxTokenBytes = 64 << 10
	// maxNumericDate is 9999-12-31T23:59:59Z. It keeps a NumericDate inside
	// the range where the conversion to int64 is defined.
	maxNumericDate = 253402300799
)

// strictB64 rejects encodings with non-zero trailing bits.
var strictB64 = base64.RawURLEncoding.Strict()

// VerifyOption configures VerifyWithKeySet.
type VerifyOption func(*verifyConfig)

type verifyConfig struct {
	ctx    context.Context
	leeway time.Duration
}

// WithLeeway tolerates clock skew of d when checking "exp", "nbf", and "iat".
// The default is zero. Negative values count as zero.
func WithLeeway(d time.Duration) VerifyOption {
	return func(c *verifyConfig) { c.leeway = max(d, 0) }
}

// WithContext sets the context for the key lookup. A JWKSClient stops waiting
// for a refresh when ctx ends. The default is context.Background.
func WithContext(ctx context.Context) VerifyOption {
	return func(c *verifyConfig) {
		if ctx != nil {
			c.ctx = ctx
		}
	}
}

// VerifyWithKeySet validates a compact JWT against ks and returns its claims.
// It is meant for tokens from an external issuer and is stricter than Verify:
//
//   - The header must name an "alg" from allowedAlgs, compared case-sensitively.
//     "none" is rejected even if listed, and an empty list rejects every token.
//   - The header must carry a "kid". The key it selects must be bound to that
//     same "alg", so a public key can never be used as an HMAC secret.
//   - Headers such as "jwk", "jku", and "x5u" are ignored, and a token with a
//     "crit" header is rejected because no extension is understood.
//   - "exp" is required. "exp", "nbf", and "iat" must be in-range numbers and
//     are checked with the leeway set by WithLeeway.
//
// "iss", "aud", and any application claim remain the caller's responsibility.
// Errors never contain token or key material. ErrKeySetUnavailable means the
// token could not be judged, and every other error means it was rejected.
func VerifyWithKeySet(token string, ks KeySet, allowedAlgs []Algorithm, opts ...VerifyOption) (*Claims, error) {
	if ks == nil {
		return nil, ErrKeySetUnavailable
	}
	cfg := verifyConfig{ctx: context.Background()}
	for _, opt := range opts {
		opt(&cfg)
	}

	header, payload, sig, err := splitToken(token)
	if err != nil {
		return nil, err
	}
	alg, kid, err := parseTokenHeader(header, allowedAlgs)
	if err != nil {
		return nil, err
	}
	signature, err := strictB64.DecodeString(sig)
	if err != nil {
		return nil, ErrTokenMalformed
	}

	key, err := ks.Lookup(cfg.ctx, kid)
	if err != nil {
		return nil, err
	}
	if key.Alg != alg {
		return nil, ErrAlgorithmMismatch
	}
	if err := verifyRaw(key, []byte(header+"."+payload), signature); err != nil {
		return nil, err
	}
	return parseVerifiedClaims(payload, cfg.leeway)
}

// splitToken cuts a compact JWT into its three segments. SplitN keeps the
// allocation constant for a token made of dots. Line breaks are refused
// because the base64 decoder would otherwise skip them.
func splitToken(token string) (header, payload, sig string, err error) {
	if len(token) > maxTokenBytes || strings.ContainsAny(token, "\r\n") {
		return "", "", "", ErrTokenMalformed
	}
	parts := strings.SplitN(token, ".", 4)
	if len(parts) != 3 {
		return "", "", "", ErrTokenMalformed
	}
	return parts[0], parts[1], parts[2], nil
}

// parseTokenHeader decodes the header into a map so that member names match
// exactly: encoding/json would let "ALG" fill a field tagged "alg".
func parseTokenHeader(segment string, allowedAlgs []Algorithm) (Algorithm, string, error) {
	data, err := strictB64.DecodeString(segment)
	if err != nil {
		return "", "", ErrTokenMalformed
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return "", "", ErrTokenMalformed
	}
	alg, err := headerString(raw, "alg")
	if err != nil {
		return "", "", err
	}
	if alg == "" || strings.EqualFold(alg, "none") {
		return "", "", ErrAlgorithmNone
	}
	if !slices.Contains(allowedAlgs, Algorithm(alg)) {
		return "", "", ErrAlgorithmNotAllowed
	}
	if _, ok := raw["crit"]; ok {
		return "", "", ErrTokenInvalid
	}
	kid, err := headerString(raw, "kid")
	if err != nil {
		return "", "", err
	}
	if kid == "" {
		return "", "", ErrKeyIDMissing
	}
	return Algorithm(alg), kid, nil
}

func headerString(raw map[string]json.RawMessage, name string) (string, error) {
	value, ok := raw[name]
	if !ok {
		return "", nil
	}
	var s string
	if err := json.Unmarshal(value, &s); err != nil {
		return "", ErrTokenMalformed
	}
	return s, nil
}

func parseVerifiedClaims(segment string, leeway time.Duration) (*Claims, error) {
	data, err := strictB64.DecodeString(segment)
	if err != nil {
		return nil, ErrTokenMalformed
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, ErrTokenMalformed
	}
	if err := checkTimeClaims(raw, nowFunc(), leeway); err != nil {
		return nil, err
	}
	return claimsFromMap(raw), nil
}

func checkTimeClaims(raw map[string]any, now time.Time, leeway time.Duration) error {
	exp, err := numericDate(raw, "exp")
	if err != nil {
		return err
	}
	if exp.IsZero() {
		return ErrTokenInvalid
	}
	if !now.Before(exp.Add(leeway)) {
		return ErrTokenExpired
	}
	for _, name := range []string{"nbf", "iat"} {
		t, err := numericDate(raw, name)
		if err != nil {
			return err
		}
		if t.After(now.Add(leeway)) {
			return ErrTokenInvalid
		}
	}
	return nil
}

// numericDate returns the zero time for an absent claim. claimsFromMap would
// silently drop a claim of the wrong type, which must not disable the check.
func numericDate(raw map[string]any, name string) (time.Time, error) {
	v, ok := raw[name]
	if !ok {
		return time.Time{}, nil
	}
	seconds, ok := v.(float64)
	if !ok || seconds < 0 || seconds > maxNumericDate {
		return time.Time{}, ErrTokenMalformed
	}
	return time.Unix(int64(seconds), 0), nil
}
