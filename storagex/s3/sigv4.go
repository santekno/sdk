package s3

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strconv"
	"strings"
	"time"
)

// AWS Signature Version 4 (SigV4), as used by S3: the canonical URI is
// URI-encoded once and is not normalized, so "/a/../b" and "//" are signed
// as written.

const (
	sigAlgorithm   = "AWS4-HMAC-SHA256"
	amzDateFormat  = "20060102T150405Z"
	scopeDate      = "20060102"
	unsignedBody   = "UNSIGNED-PAYLOAD"
	emptyBodyHash  = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	maxPresignTTL  = 7 * 24 * time.Hour
	scopeTerminate = "aws4_request"
)

// pair is one header or query parameter.
type pair struct{ k, v string }

// signer computes SigV4 signatures for one credential, region, and service.
type signer struct {
	accessKeyID     string
	secretAccessKey string
	region          string
	service         string
}

// uriEncode percent-encodes every byte of s except the unreserved characters
// A-Z a-z 0-9 - . _ ~, with uppercase hex; '/' is kept when keepSlash is set.
func uriEncode(s string, keepSlash bool) string {
	const hexDigits = "0123456789ABCDEF"
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case 'A' <= c && c <= 'Z', 'a' <= c && c <= 'z', '0' <= c && c <= '9',
			c == '-', c == '.', c == '_', c == '~', keepSlash && c == '/':
			b.WriteByte(c)
		default:
			b.WriteByte('%')
			b.WriteByte(hexDigits[c>>4])
			b.WriteByte(hexDigits[c&0x0f])
		}
	}
	return b.String()
}

// canonicalQuery encodes each name and value and sorts by name, then value.
func canonicalQuery(params []pair) string {
	enc := make([]pair, len(params))
	for i, p := range params {
		enc[i] = pair{uriEncode(p.k, false), uriEncode(p.v, false)}
	}
	sort.Slice(enc, func(i, j int) bool {
		if enc[i].k != enc[j].k {
			return enc[i].k < enc[j].k
		}
		return enc[i].v < enc[j].v
	})
	parts := make([]string, len(enc))
	for i, p := range enc {
		parts[i] = p.k + "=" + p.v
	}
	return strings.Join(parts, "&")
}

// canonicalHeaders lowercases names, trims values and collapses inner runs of
// spaces, joins repeated names with ',' in their original order, and sorts by
// name. It returns the canonical header block and the signed header list.
func canonicalHeaders(headers []pair) (string, string) {
	values := map[string][]string{}
	names := make([]string, 0, len(headers))
	for _, h := range headers {
		k := strings.ToLower(strings.TrimSpace(h.k))
		if _, ok := values[k]; !ok {
			names = append(names, k)
		}
		values[k] = append(values[k], strings.Join(strings.Fields(h.v), " "))
	}
	sort.Strings(names)
	var b strings.Builder
	for _, k := range names {
		b.WriteString(k)
		b.WriteByte(':')
		b.WriteString(strings.Join(values[k], ","))
		b.WriteByte('\n')
	}
	return b.String(), strings.Join(names, ";")
}

// canonicalRequest builds the SigV4 canonical request. escapedPath must
// already be URI-encoded (see uriEncode).
func canonicalRequest(method, escapedPath string, query, headers []pair, payloadHash string) (creq, signedHeaders string) {
	ch, sh := canonicalHeaders(headers)
	creq = strings.Join([]string{method, escapedPath, canonicalQuery(query), ch, sh, payloadHash}, "\n")
	return creq, sh
}

func (s signer) scope(t time.Time) string {
	return t.UTC().Format(scopeDate) + "/" + s.region + "/" + s.service + "/" + scopeTerminate
}

func (s signer) stringToSign(t time.Time, creq string) string {
	h := sha256.Sum256([]byte(creq))
	return sigAlgorithm + "\n" + t.UTC().Format(amzDateFormat) + "\n" + s.scope(t) + "\n" + hex.EncodeToString(h[:])
}

func hmacSHA256(key []byte, data string) []byte {
	m := hmac.New(sha256.New, key)
	m.Write([]byte(data))
	return m.Sum(nil)
}

// signature derives the signing key for t and signs stringToSign.
func (s signer) signature(t time.Time, stringToSign string) string {
	k := hmacSHA256([]byte("AWS4"+s.secretAccessKey), t.UTC().Format(scopeDate))
	k = hmacSHA256(k, s.region)
	k = hmacSHA256(k, s.service)
	k = hmacSHA256(k, scopeTerminate)
	return hex.EncodeToString(hmacSHA256(k, stringToSign))
}

// authorization signs a request for the Authorization header. headers must
// include host and x-amz-date (and x-amz-content-sha256 for S3).
func (s signer) authorization(method, escapedPath string, query, headers []pair, payloadHash string, t time.Time) string {
	creq, signed := canonicalRequest(method, escapedPath, query, headers, payloadHash)
	sig := s.signature(t, s.stringToSign(t, creq))
	return sigAlgorithm + " Credential=" + s.accessKeyID + "/" + s.scope(t) +
		", SignedHeaders=" + signed + ", Signature=" + sig
}

// presignQuery returns the query parameters of a SigV4 query-signed request
// (X-Amz-Algorithm ... X-Amz-Signature) for the given request parts. query
// holds the request's own parameters, which are signed as well.
func (s signer) presignQuery(method, escapedPath string, query, headers []pair, payloadHash string, t time.Time, ttl time.Duration) []pair {
	_, signed := canonicalHeaders(headers)
	auth := []pair{
		{"X-Amz-Algorithm", sigAlgorithm},
		{"X-Amz-Credential", s.accessKeyID + "/" + s.scope(t)},
		{"X-Amz-Date", t.UTC().Format(amzDateFormat)},
		{"X-Amz-Expires", formatSeconds(ttl)},
		{"X-Amz-SignedHeaders", signed},
	}
	all := append(append([]pair{}, query...), auth...)
	creq, _ := canonicalRequest(method, escapedPath, all, headers, payloadHash)
	sig := s.signature(t, s.stringToSign(t, creq))
	return append(all, pair{"X-Amz-Signature", sig})
}

func formatSeconds(d time.Duration) string {
	return strconv.FormatInt(int64(d/time.Second), 10)
}

func sha256Hex(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}
