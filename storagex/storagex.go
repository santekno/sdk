// Package storagex provides a Storage abstraction for binary blobs (T016,
// Constitution II). MemStorage is provided for tests, LocalStorage writes to
// the local filesystem, and the storagex/s3 subpackage talks to S3-compatible
// object storage (AWS S3, Supabase Storage, MinIO).
//
// Every backend implements Storage. Inspector (Stat, List) and Presigner
// (PresignPut) are optional interfaces; check for them with a type assertion.
// Keys are slash-separated relative paths validated by ValidateKey, and
// backends report missing objects with ErrNotFound and refused overwrites
// (WithNoOverwrite) with ErrAlreadyExists. Errors never echo keys, URLs, or
// credentials.
package storagex

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"time"
)

// Storage stores and retrieves binary blobs by key.
type Storage interface {
	// Put writes r under key and returns a URL (or path) to the stored blob.
	Put(ctx context.Context, key, contentType string, r io.Reader) (url string, err error)
	// Get returns a reader for the blob stored under key.
	Get(ctx context.Context, key string) (io.ReadCloser, error)
	// Delete removes the blob stored under key.
	Delete(ctx context.Context, key string) error
}

var (
	// ErrNotFound is matched (errors.Is) by errors for a key with no object.
	ErrNotFound = errors.New("storagex: object not found")
	// ErrAlreadyExists is matched by Put errors when the storage was created
	// with WithNoOverwrite and an object already exists under the key.
	ErrAlreadyExists = errors.New("storagex: object already exists")
	// ErrInvalidKey is returned when a key or list prefix is rejected by
	// ValidateKey or ValidatePrefix. The error never echoes the key.
	ErrInvalidKey = errors.New("storagex: invalid key")
)

// ObjectInfo describes a stored object. Fields a backend cannot provide are
// left empty: LocalStorage has no ContentType or ETag, and List results from
// S3 have no ContentType.
type ObjectInfo struct {
	Key          string
	Size         int64
	ContentType  string
	ETag         string // opaque; compare only values from the same backend
	LastModified time.Time
}

// MaxListLimit is the largest page List returns.
const MaxListLimit = 1000

// ListOptions selects one page of a listing.
type ListOptions struct {
	// Prefix limits the listing to keys that start with it (see ValidatePrefix).
	Prefix string
	// Token continues a listing; pass ListPage.NextToken from the previous page.
	Token string
	// Limit is the page size: values <= 0 or > MaxListLimit mean MaxListLimit.
	Limit int
}

// PageLimit returns the effective page size for o.Limit.
func (o ListOptions) PageLimit() int {
	if o.Limit <= 0 || o.Limit > MaxListLimit {
		return MaxListLimit
	}
	return o.Limit
}

// ListPage is one page of a listing, sorted by key.
type ListPage struct {
	Objects []ObjectInfo
	// NextToken is empty on the last page.
	NextToken string
}

// Inspector is implemented by backends that can describe and list objects.
type Inspector interface {
	// Stat describes the object under key, or returns an error matching
	// ErrNotFound.
	Stat(ctx context.Context, key string) (ObjectInfo, error)
	// List returns one page of objects whose key starts with opts.Prefix.
	List(ctx context.Context, opts ListOptions) (ListPage, error)
}

// PresignOptions describes the one upload a presigned request allows.
type PresignOptions struct {
	// ContentType is the exact Content-Type the client must send.
	ContentType string
	// ContentLength is the exact body size in bytes the client must send; it
	// must be greater than zero. Browsers set Content-Length from the body, so
	// pass the size the client declared and check it against your limits
	// before presigning.
	ContentLength int64
	// TTL is how long the request stays valid. Backends reject values they
	// cannot honor instead of shortening them.
	TTL time.Duration
}

// Presigner is implemented by backends that can hand out a URL that lets a
// client without credentials upload one object directly.
type Presigner interface {
	// PresignPut returns a request that uploads one object under key with
	// exactly opts.ContentType and opts.ContentLength, valid for opts.TTL.
	PresignPut(ctx context.Context, key string, opts PresignOptions) (PresignedRequest, error)
}

// PresignedRequest is a request a client can send without credentials.
//
// URL is a bearer credential until ExpiresAt: hand it only to the client that
// uploads and never log it. String, GoString, and LogValue redact the query,
// so fmt verbs and log/slog do not print the signature.
type PresignedRequest struct {
	Method string
	URL    string
	// Header lists headers the client must send exactly as given; they are
	// part of the signature.
	Header    http.Header
	ExpiresAt time.Time
}

// RedactURL returns raw without its query and fragment (replaced by
// "?REDACTED" when a query was present) and without user info. It returns
// "REDACTED" when raw cannot be parsed.
func RedactURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return "REDACTED"
	}
	hadQuery := u.RawQuery != "" || u.ForceQuery
	u.User, u.RawQuery, u.ForceQuery, u.Fragment, u.RawFragment = nil, "", false, "", ""
	s := u.String()
	if hadQuery {
		s += "?REDACTED"
	}
	return s
}

// String returns the method, the URL without its query, and the expiry.
func (p PresignedRequest) String() string {
	return fmt.Sprintf("%s %s (expires %s)", p.Method, RedactURL(p.URL), p.ExpiresAt.UTC().Format(time.RFC3339))
}

// GoString is the same as String, so %#v does not print the signature.
func (p PresignedRequest) GoString() string { return p.String() }

// LogValue implements slog.LogValuer without the URL query.
func (p PresignedRequest) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("method", p.Method),
		slog.String("url", RedactURL(p.URL)),
		slog.Time("expires_at", p.ExpiresAt),
	)
}

// Options holds settings shared by every backend constructor.
type Options struct {
	// NoOverwrite makes Put fail with ErrAlreadyExists instead of replacing an
	// existing object.
	NoOverwrite bool
}

// Option configures a backend constructor.
type Option func(*Options)

// WithNoOverwrite makes Put refuse to replace an existing object: it returns
// an error matching ErrAlreadyExists and leaves the stored object unchanged.
func WithNoOverwrite() Option { return func(o *Options) { o.NoOverwrite = true } }

// NewOptions applies opts in order. Backend implementations call it from
// their constructors.
func NewOptions(opts ...Option) Options {
	var o Options
	for _, opt := range opts {
		if opt != nil {
			opt(&o)
		}
	}
	return o
}

var (
	_ Storage   = (*MemStorage)(nil)
	_ Inspector = (*MemStorage)(nil)
	_ Storage   = (*LocalStorage)(nil)
	_ Inspector = (*LocalStorage)(nil)
)
