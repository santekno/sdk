// Package s3 is a storagex backend for S3-compatible object storage (AWS S3,
// Supabase Storage S3 connection, MinIO), written with the standard library
// only.
//
// A Client serves one bucket with one set of credentials and implements
// storagex.Storage, storagex.Inspector, and storagex.Presigner. Server calls
// are signed with AWS Signature Version 4 in the Authorization header and
// carry the SHA-256 of the body; PresignPut signs with query parameters
// (UNSIGNED-PAYLOAD) so a browser can upload one object without credentials.
// Requests use path-style URLs: <Endpoint>/<Bucket>/<key>.
//
// Security properties:
//   - Errors (see Error) never contain the request URL, the object key, the
//     Authorization header, credentials, or the server's error message.
//   - Config, Client, and storagex.PresignedRequest redact secrets and URL
//     queries when formatted with fmt or log/slog.
//   - Redirects are never followed; a 3xx response is an error.
//   - Error and listing response bodies are read up to a fixed limit.
//   - Endpoint must be https, except for loopback hosts (local emulators).
//
// Callers bound the duration of each call with the context; the default HTTP
// client only limits connection setup and the wait for response headers, so
// large transfers are not cut off.
package s3

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"hash"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/santekno/sdk/storagex"
)

const (
	maxErrorBody = 64 << 10 // error responses
	maxListBody  = 8 << 20  // one ListObjectsV2 page of 1000 keys
)

// Config configures one Client. Each Client has its own credentials, so
// separate keys (for example one for media and one for backups) mean separate
// Clients.
type Config struct {
	// Endpoint is the S3 base URL, e.g.
	// "https://<project>.supabase.co/storage/v1/s3" or "http://127.0.0.1:54321/storage/v1/s3".
	// It may include a path; it must not include user info, a query, or a
	// fragment. http is accepted only for loopback hosts.
	Endpoint string
	// Region is the signing region (e.g. "ap-southeast-1"; "local" for the
	// Supabase CLI).
	Region string
	// Bucket is the bucket name (letters, digits, '.', '-', '_').
	Bucket string
	// AccessKeyID and SecretAccessKey are the S3 credentials.
	AccessKeyID     string
	SecretAccessKey string
	// HTTPClient is optional. The Client uses a copy whose CheckRedirect
	// refuses redirects. The default has a 10 s dial and TLS handshake timeout
	// and a 30 s response header timeout.
	HTTPClient *http.Client
	// TempDir is where Put buffers bodies of unknown length (default
	// os.TempDir()).
	TempDir string
	// MaxPresignTTL is the longest TTL PresignPut accepts; longer TTLs are
	// rejected. Zero means 7 days, the SigV4 limit and the largest value
	// allowed. Set it to your policy (for example 10 minutes for browser
	// uploads) so a caller bug cannot mint long-lived upload URLs.
	MaxPresignTTL time.Duration
}

// String returns the configuration without credentials.
func (c Config) String() string {
	return fmt.Sprintf("s3.Config{Endpoint:%q, Region:%q, Bucket:%q, AccessKeyID:REDACTED, SecretAccessKey:REDACTED}",
		storagex.RedactURL(c.Endpoint), c.Region, c.Bucket)
}

// GoString is the same as String.
func (c Config) GoString() string { return c.String() }

// LogValue implements slog.LogValuer without credentials.
func (c Config) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("endpoint", storagex.RedactURL(c.Endpoint)),
		slog.String("region", c.Region),
		slog.String("bucket", c.Bucket),
	)
}

// Client is an S3-compatible storagex backend for one bucket. It is safe for
// concurrent use.
type Client struct {
	scheme   string
	host     string // canonical host[:port], default port removed
	basePath string // decoded endpoint path without trailing slash
	bucket   string
	sign     signer
	hc       *http.Client
	opts     storagex.Options
	tempDir  string
	now      func() time.Time

	maxPresignTTL time.Duration
}

var (
	_ storagex.Storage   = (*Client)(nil)
	_ storagex.Inspector = (*Client)(nil)
	_ storagex.Presigner = (*Client)(nil)
)

// New validates cfg and returns a Client. Errors match ErrInvalidConfig and
// name the offending field without its value.
func New(cfg Config, opts ...storagex.Option) (*Client, error) {
	u, err := validateConfig(cfg)
	if err != nil {
		return nil, err
	}
	hc := defaultHTTPClient()
	if cfg.HTTPClient != nil {
		c := *cfg.HTTPClient
		hc = &c
	}
	hc.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }

	return &Client{
		scheme:   u.Scheme,
		host:     canonicalHost(u),
		basePath: strings.TrimSuffix(u.Path, "/"),
		bucket:   cfg.Bucket,
		sign: signer{
			accessKeyID:     cfg.AccessKeyID,
			secretAccessKey: cfg.SecretAccessKey,
			region:          cfg.Region,
			service:         "s3",
		},
		hc:      hc,
		opts:    storagex.NewOptions(opts...),
		tempDir: cfg.TempDir,
		now:     time.Now,
		maxPresignTTL: func() time.Duration {
			if cfg.MaxPresignTTL == 0 {
				return maxPresignTTL
			}
			return cfg.MaxPresignTTL
		}(),
	}, nil
}

func invalidConfig(field, why string) error {
	return fmt.Errorf("%w: %s %s", ErrInvalidConfig, field, why)
}

// validateConfig checks cfg and returns the parsed endpoint.
func validateConfig(cfg Config) (*url.URL, error) {
	u, err := validateEndpoint(cfg.Endpoint)
	if err != nil {
		return nil, err
	}
	switch {
	case !validToken(cfg.Region, "-_"):
		return nil, invalidConfig("Region", "must be letters, digits, '-' or '_'")
	case len(cfg.Bucket) > 255 || cfg.Bucket == "." || cfg.Bucket == ".." || !validToken(cfg.Bucket, ".-_"):
		return nil, invalidConfig("Bucket", "must be letters, digits, '.', '-' or '_'")
	case !validToken(cfg.AccessKeyID, ".-_+="):
		return nil, invalidConfig("AccessKeyID", "is empty or has unsupported characters")
	case cfg.SecretAccessKey == "":
		return nil, invalidConfig("SecretAccessKey", "is empty")
	case cfg.MaxPresignTTL != 0 && (cfg.MaxPresignTTL < time.Second || cfg.MaxPresignTTL > maxPresignTTL):
		return nil, invalidConfig("MaxPresignTTL", "must be between 1s and 7 days")
	}
	return u, nil
}

func validateEndpoint(endpoint string) (*url.URL, error) {
	u, err := url.Parse(endpoint)
	if err != nil || u.Host == "" || u.Opaque != "" {
		return nil, invalidConfig("Endpoint", "must be an absolute URL")
	}
	if u.User != nil || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" {
		return nil, invalidConfig("Endpoint", "must not have user info, a query, or a fragment")
	}
	switch u.Scheme {
	case "https":
		return u, nil
	case "http":
		if isLoopback(u.Hostname()) {
			return u, nil
		}
		return nil, invalidConfig("Endpoint", "must use https (http only for loopback hosts)")
	default:
		return nil, invalidConfig("Endpoint", "must use https")
	}
}

// String describes the Client without credentials. It has a value receiver
// so that formatting a dereferenced Client does not print its fields.
func (c Client) String() string {
	return "s3.Client{" + c.scheme + "://" + c.host + c.basePath + "/" + c.bucket + "}"
}

// GoString is the same as String.
func (c Client) GoString() string { return c.String() }

func defaultHTTPClient() *http.Client {
	return &http.Client{Transport: &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           (&net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          32,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,
		ExpectContinueTimeout: time.Second,
	}}
}

func isLoopback(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func canonicalHost(u *url.URL) string {
	port := u.Port()
	if (u.Scheme == "https" && port == "443") || (u.Scheme == "http" && port == "80") {
		return strings.ToLower(u.Hostname())
	}
	return strings.ToLower(u.Host)
}

// validToken reports whether s is non-empty and made of ASCII letters,
// digits, and the given extra characters.
func validToken(s, extra string) bool {
	if s == "" || len(s) > 256 {
		return false
	}
	for i := 0; i < len(s); i++ {
		if !isAlnum(s[i]) && strings.IndexByte(extra, s[i]) < 0 {
			return false
		}
	}
	return true
}

func isAlnum(c byte) bool {
	return 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z' || '0' <= c && c <= '9'
}

// objectPath returns the decoded path of key (or of the bucket when key is
// empty).
func (c *Client) objectPath(key string) string {
	p := c.basePath + "/" + c.bucket
	if key != "" {
		p += "/" + key
	}
	return p
}

// request is one signed server call.
type request struct {
	op, method  string
	key         string // "" for bucket operations
	query       []pair
	headers     []pair // extra signed headers
	body        io.Reader
	size        int64
	payloadHash string
}

func (c *Client) do(ctx context.Context, r request) (*http.Response, error) {
	t := c.now().UTC()
	decoded := c.objectPath(r.key)
	escaped := uriEncode(decoded, true)
	headers := append([]pair{
		{"Host", c.host},
		{"X-Amz-Date", t.Format(amzDateFormat)},
		{"X-Amz-Content-Sha256", r.payloadHash},
	}, r.headers...)
	auth := c.sign.authorization(r.method, escaped, r.query, headers, r.payloadHash, t)

	u := &url.URL{Scheme: c.scheme, Host: c.host, Path: decoded, RawPath: escaped, RawQuery: canonicalQuery(r.query)}
	body := r.body
	if r.size == 0 {
		body = http.NoBody
	}
	req, err := http.NewRequestWithContext(ctx, r.method, u.String(), body)
	if err != nil {
		return nil, &Error{Op: r.op, detail: "cannot build request"}
	}
	req.ContentLength = r.size
	req.Host = c.host
	for _, h := range headers[1:] {
		req.Header.Set(h.k, h.v)
	}
	req.Header.Set("Authorization", auth)
	// Ask for the stored bytes as they are; this also stops net/http from
	// transparently decompressing a gzip-encoded object.
	req.Header.Set("Accept-Encoding", "identity")
	resp, err := c.hc.Do(req)
	if err != nil {
		return nil, transportError(r.op, err)
	}
	return resp, nil
}

// statusError builds the error for an unexpected response, reading at most
// maxErrorBody bytes of its body. The caller closes the body.
func statusError(op string, resp *http.Response) *Error {
	e := &Error{Op: op, StatusCode: resp.StatusCode}
	if resp.StatusCode >= 300 && resp.StatusCode < 400 {
		e.detail = "redirect refused"
		return e
	}
	var body struct {
		Code string `xml:"Code"`
	}
	if b, err := io.ReadAll(io.LimitReader(resp.Body, maxErrorBody)); err == nil && len(b) > 0 {
		if xml.Unmarshal(b, &body) == nil {
			e.Code = safeCode(body.Code)
		}
	}
	return e
}

// drain reads what is left of a small body so the connection can be reused.
func drain(r io.Reader) {
	_, _ = io.Copy(io.Discard, io.LimitReader(r, maxErrorBody))
}

// objectURL is the credential-free URL of key.
func (c *Client) objectURL(key string) string {
	return c.scheme + "://" + c.host + uriEncode(c.objectPath(key), true)
}

// Put uploads r under key and returns the object URL (no credentials). A body
// whose length cannot be determined by seeking is first copied to a temporary
// file in Config.TempDir to compute its length and SHA-256, not buffered in
// memory. That copy has no size limit: bound untrusted readers yourself (for
// example with io.LimitReader). The temporary file is removed when Put
// returns, but a process killed during Put leaves a "storagex-s3-*" file in
// TempDir. With storagex.WithNoOverwrite the request carries
// "If-None-Match: *" and a 412 response returns an error matching
// storagex.ErrAlreadyExists; servers that ignore the header overwrite.
func (c *Client) Put(ctx context.Context, key, contentType string, r io.Reader) (string, error) {
	if err := storagex.ValidateKey(key); err != nil {
		return "", err
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	if !validHeaderValue(contentType) {
		return "", fmt.Errorf("%w: content type", ErrInvalidRequest)
	}
	body, size, sum, cleanup, err := c.prepareBody(ctx, r)
	if err != nil {
		return "", err
	}
	defer cleanup()

	headers := []pair{{"Content-Type", contentType}}
	if c.opts.NoOverwrite {
		headers = append(headers, pair{"If-None-Match", "*"})
	}
	resp, err := c.do(ctx, request{op: "put", method: http.MethodPut, key: key, headers: headers, body: body, size: size, payloadHash: sum})
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	defer drain(resp.Body)
	switch resp.StatusCode {
	case http.StatusOK, http.StatusCreated, http.StatusNoContent:
		return c.objectURL(key), nil
	case http.StatusPreconditionFailed:
		e := statusError("put", resp)
		e.sentinel, e.detail = storagex.ErrAlreadyExists, "object already exists"
		return "", e
	default:
		return "", statusError("put", resp)
	}
}

// prepareBody returns a body reader positioned at its start, its length, and
// its hex SHA-256. Seekable readers are hashed in place; other readers are
// spooled to a temporary file that cleanup removes.
func (c *Client) prepareBody(ctx context.Context, r io.Reader) (io.Reader, int64, string, func(), error) {
	noop := func() {}
	if r == nil {
		return http.NoBody, 0, emptyBodyHash, noop, nil
	}
	rs, ok := r.(io.ReadSeeker)
	if !ok {
		return c.spool(ctx, r)
	}
	start, err := rs.Seek(0, io.SeekCurrent)
	if err != nil {
		return c.spool(ctx, r) // not seekable after all (a pipe or terminal)
	}
	end, err := rs.Seek(0, io.SeekEnd)
	if _, serr := rs.Seek(start, io.SeekStart); serr != nil {
		return nil, 0, "", noop, serr
	}
	if err != nil || end < start {
		return c.spool(ctx, r)
	}
	h := sha256.New()
	if _, err := io.Copy(h, ctxReader{ctx, io.LimitReader(rs, end-start)}); err != nil {
		return nil, 0, "", noop, err
	}
	if _, err := rs.Seek(start, io.SeekStart); err != nil {
		return nil, 0, "", noop, err
	}
	return io.LimitReader(rs, end-start), end - start, hexSum(h), noop, nil
}

func (c *Client) spool(ctx context.Context, r io.Reader) (io.Reader, int64, string, func(), error) {
	f, err := os.CreateTemp(c.tempDir, "storagex-s3-*")
	if err != nil {
		return nil, 0, "", func() {}, err
	}
	cleanup := func() {
		_ = f.Close()
		_ = os.Remove(f.Name())
	}
	h := sha256.New()
	n, err := io.Copy(io.MultiWriter(f, h), ctxReader{ctx, r})
	if err == nil {
		_, err = f.Seek(0, io.SeekStart)
	}
	if err != nil {
		cleanup()
		return nil, 0, "", func() {}, err
	}
	return f, n, hexSum(h), cleanup, nil
}

func hexSum(h hash.Hash) string { return hex.EncodeToString(h.Sum(nil)) }

// ctxReader stops reading once ctx is done.
type ctxReader struct {
	ctx context.Context
	r   io.Reader
}

func (c ctxReader) Read(p []byte) (int, error) {
	if err := c.ctx.Err(); err != nil {
		return 0, err
	}
	return c.r.Read(p)
}

func validHeaderValue(v string) bool {
	if v == "" || len(v) > 256 {
		return false
	}
	for i := 0; i < len(v); i++ {
		if v[i] < 0x20 || v[i] > 0x7e {
			return false
		}
	}
	return true
}

// Get returns the object body; the caller must close it. A missing object
// returns an error matching storagex.ErrNotFound. The body is not size-limited:
// call Stat first or wrap it in io.LimitReader for untrusted objects.
func (c *Client) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	if err := storagex.ValidateKey(key); err != nil {
		return nil, err
	}
	resp, err := c.do(ctx, request{op: "get", method: http.MethodGet, key: key, payloadHash: emptyBodyHash})
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusOK {
		return resp.Body, nil
	}
	defer resp.Body.Close()
	defer drain(resp.Body)
	e := statusError("get", resp)
	if missingKey(e) {
		e.sentinel, e.detail = storagex.ErrNotFound, "object not found"
	}
	return nil, e
}

// missingKey reports whether a status error means the object does not exist:
// 404 with no S3 error code (HEAD, some servers) or NoSuchKey. Other 404s,
// such as NoSuchBucket, are configuration errors and are reported as such.
func missingKey(e *Error) bool {
	return e.StatusCode == http.StatusNotFound && (e.Code == "" || e.Code == "NoSuchKey")
}

func notFoundError(op string, resp *http.Response) *Error {
	e := statusError(op, resp)
	e.sentinel, e.detail = storagex.ErrNotFound, "object not found"
	return e
}

// Delete removes the object. Deleting a missing object succeeds, as in S3.
func (c *Client) Delete(ctx context.Context, key string) error {
	if err := storagex.ValidateKey(key); err != nil {
		return err
	}
	resp, err := c.do(ctx, request{op: "delete", method: http.MethodDelete, key: key, payloadHash: emptyBodyHash})
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	defer drain(resp.Body)
	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNoContent {
		return nil
	}
	e := statusError("delete", resp)
	if missingKey(e) {
		return nil
	}
	return e
}

// Stat describes the object with a HEAD request. A missing object returns an
// error matching storagex.ErrNotFound.
func (c *Client) Stat(ctx context.Context, key string) (storagex.ObjectInfo, error) {
	if err := storagex.ValidateKey(key); err != nil {
		return storagex.ObjectInfo{}, err
	}
	resp, err := c.do(ctx, request{op: "stat", method: http.MethodHead, key: key, payloadHash: emptyBodyHash})
	if err != nil {
		return storagex.ObjectInfo{}, err
	}
	defer resp.Body.Close()
	defer drain(resp.Body)
	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusNotFound:
		return storagex.ObjectInfo{}, notFoundError("stat", resp)
	default:
		return storagex.ObjectInfo{}, statusError("stat", resp)
	}
	size := resp.ContentLength
	if size < 0 {
		size, err = strconv.ParseInt(resp.Header.Get("Content-Length"), 10, 64)
	}
	if err != nil || size < 0 {
		return storagex.ObjectInfo{}, &Error{Op: "stat", StatusCode: resp.StatusCode, detail: "missing or invalid Content-Length"}
	}
	info := storagex.ObjectInfo{
		Key:         key,
		Size:        size,
		ContentType: resp.Header.Get("Content-Type"),
		ETag:        strings.Trim(resp.Header.Get("ETag"), `"`),
	}
	if lm, err := http.ParseTime(resp.Header.Get("Last-Modified")); err == nil {
		info.LastModified = lm.UTC()
	}
	return info, nil
}

type listBucketResult struct {
	IsTruncated           bool   `xml:"IsTruncated"`
	NextContinuationToken string `xml:"NextContinuationToken"`
	Contents              []struct {
		Key          string `xml:"Key"`
		LastModified string `xml:"LastModified"`
		ETag         string `xml:"ETag"`
		Size         int64  `xml:"Size"`
	} `xml:"Contents"`
}

// List returns one page of objects whose key starts with opts.Prefix, using
// ListObjectsV2. NextToken is the server's continuation token. Objects have no
// ContentType. The response body is limited to 8 MiB.
func (c *Client) List(ctx context.Context, opts storagex.ListOptions) (storagex.ListPage, error) {
	if err := storagex.ValidatePrefix(opts.Prefix); err != nil {
		return storagex.ListPage{}, err
	}
	query := []pair{{"list-type", "2"}, {"max-keys", strconv.Itoa(opts.PageLimit())}}
	if opts.Prefix != "" {
		query = append(query, pair{"prefix", opts.Prefix})
	}
	if opts.Token != "" {
		query = append(query, pair{"continuation-token", opts.Token})
	}
	resp, err := c.do(ctx, request{op: "list", method: http.MethodGet, query: query, payloadHash: emptyBodyHash})
	if err != nil {
		return storagex.ListPage{}, err
	}
	defer resp.Body.Close()
	defer drain(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return storagex.ListPage{}, statusError("list", resp)
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, maxListBody+1))
	if err != nil {
		return storagex.ListPage{}, transportError("list", err)
	}
	if len(b) > maxListBody {
		return storagex.ListPage{}, &Error{Op: "list", StatusCode: resp.StatusCode, detail: "response too large"}
	}
	var res listBucketResult
	if err := xml.NewDecoder(bytes.NewReader(b)).Decode(&res); err != nil {
		return storagex.ListPage{}, &Error{Op: "list", StatusCode: resp.StatusCode, detail: "malformed response"}
	}
	page := storagex.ListPage{Objects: make([]storagex.ObjectInfo, 0, len(res.Contents))}
	for _, o := range res.Contents {
		info := storagex.ObjectInfo{Key: o.Key, Size: o.Size, ETag: strings.Trim(o.ETag, `"`)}
		if lm, err := time.Parse(time.RFC3339Nano, o.LastModified); err == nil {
			info.LastModified = lm.UTC()
		}
		page.Objects = append(page.Objects, info)
	}
	if res.IsTruncated {
		if res.NextContinuationToken == "" || res.NextContinuationToken == opts.Token {
			return storagex.ListPage{}, &Error{Op: "list", StatusCode: resp.StatusCode, detail: "truncated response without a new continuation token"}
		}
		page.NextToken = res.NextContinuationToken
	}
	return page, nil
}

// PresignPut returns a PUT request for key that a client without credentials
// can send until ExpiresAt. It signs the host, Content-Type, and
// Content-Length (and "If-None-Match: *" with storagex.WithNoOverwrite), so
// the client must send exactly the headers in PresignedRequest.Header and a
// body of exactly opts.ContentLength bytes; any other type or size is refused
// by the server. The body itself is not signed (UNSIGNED-PAYLOAD), so check
// the declared size against your limits before presigning and verify the
// uploaded content afterwards.
//
// opts.ContentLength must be greater than zero, and opts.TTL at least 1
// second and at most Config.MaxPresignTTL: longer TTLs are rejected with
// ErrInvalidRequest, never shortened. No request is made; ctx is unused.
//
// The returned URL is a bearer credential: never log it (its String and
// LogValue redact the query).
func (c *Client) PresignPut(_ context.Context, key string, opts storagex.PresignOptions) (storagex.PresignedRequest, error) {
	if err := storagex.ValidateKey(key); err != nil {
		return storagex.PresignedRequest{}, err
	}
	switch {
	case !validHeaderValue(opts.ContentType):
		return storagex.PresignedRequest{}, fmt.Errorf("%w: content type", ErrInvalidRequest)
	case opts.ContentLength <= 0:
		return storagex.PresignedRequest{}, fmt.Errorf("%w: content length must be greater than zero", ErrInvalidRequest)
	case opts.TTL < time.Second || opts.TTL > c.maxPresignTTL:
		return storagex.PresignedRequest{}, fmt.Errorf("%w: ttl must be between 1s and the configured maximum", ErrInvalidRequest)
	}
	ttl := opts.TTL.Truncate(time.Second)
	t := c.now().UTC()
	headers := []pair{
		{"Host", c.host},
		{"Content-Length", strconv.FormatInt(opts.ContentLength, 10)},
		{"Content-Type", opts.ContentType},
	}
	if c.opts.NoOverwrite {
		headers = append(headers, pair{"If-None-Match", "*"})
	}
	escaped := uriEncode(c.objectPath(key), true)
	params := c.sign.presignQuery(http.MethodPut, escaped, nil, headers, unsignedBody, t, ttl)

	h := http.Header{}
	for _, p := range headers[1:] {
		h.Set(p.k, p.v)
	}
	return storagex.PresignedRequest{
		Method:    http.MethodPut,
		URL:       c.scheme + "://" + c.host + escaped + "?" + canonicalQuery(params),
		Header:    h,
		ExpiresAt: t.Add(ttl),
	}, nil
}
