package jwtx

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	jwksFetchTimeout = 5 * time.Second
	// jwksDefaultTTL is also the ceiling: a longer max-age does not extend it.
	jwksDefaultTTL = 10 * time.Minute
	// jwksMinRefreshInterval limits fetches of every kind, so that tokens with
	// random "kid" values cannot turn into outbound traffic.
	jwksMinRefreshInterval = 30 * time.Second
	jwksDefaultMaxStale    = 24 * time.Hour
)

// JWKSClient is a KeySet backed by a remote JWKS document. It is safe for
// concurrent use and makes no request until the first Lookup.
//
// Keys are cached for 10 minutes, or for the response's Cache-Control max-age
// when that is shorter. An expired or unknown "kid" triggers a refresh, but the
// client fetches at most once per 30 seconds and concurrent callers share one
// fetch. A successful refresh replaces the whole set, so a key the issuer
// removed stops working at once.
//
// When a refresh fails the old keys stay in use, but only for 24 hours after
// the start of the last successful fetch (see WithMaxStale). Past that limit
// every Lookup returns ErrKeySetUnavailable until a refresh succeeds, so that
// an attacker who blocks the JWKS endpoint cannot keep a revoked key alive.
//
// Each fetch is limited to 5 seconds and 64 KiB, and redirects are refused.
type JWKSClient struct {
	url           string
	http          *http.Client
	allowLoopback bool
	timeout       time.Duration
	maxStale      time.Duration
	now           func() time.Time

	mu          sync.Mutex
	keys        *StaticKeySet // nil until the first successful fetch
	expires     time.Time
	lastAttempt time.Time
	lastSuccess time.Time // start of the last successful fetch
	lastErr     error
	inflight    chan struct{} // non-nil while a fetch runs, closed when it ends
}

// JWKSOption configures a JWKSClient.
type JWKSOption func(*JWKSClient)

// WithHTTPClient makes the client use hc's transport, for example to trust a
// private CA. A copy is taken and the redirect refusal and fetch timeout still
// apply. Nothing stops a transport from weakening TLS: that stays with the
// caller.
func WithHTTPClient(hc *http.Client) JWKSOption {
	return func(c *JWKSClient) {
		if hc != nil {
			c.http = withoutRedirects(hc)
		}
	}
}

// WithInsecureLoopback additionally accepts plain http URLs whose host is
// loopback, for a local issuer in development. Other hosts still need https.
func WithInsecureLoopback() JWKSOption {
	return func(c *JWKSClient) { c.allowLoopback = true }
}

// WithMaxStale sets how long after the last successful fetch keys may still be
// used while refreshes fail. The default is 24 hours, and a key exactly d old is
// still accepted. NewJWKSClient rejects d <= 0, so the limit cannot be switched
// off by accident. Below 30 seconds, the refresh limit leaves gaps in which even
// a healthy issuer yields ErrKeySetUnavailable.
func WithMaxStale(d time.Duration) JWKSOption {
	return func(c *JWKSClient) { c.maxStale = d }
}

// NewJWKSClient returns a client for the JWKS document at rawURL. It returns
// ErrJWKSURLInvalid unless rawURL is an absolute https URL without credentials,
// and ErrMaxStaleInvalid when WithMaxStale received a value that is not positive.
func NewJWKSClient(rawURL string, opts ...JWKSOption) (*JWKSClient, error) {
	c := &JWKSClient{
		url:      rawURL,
		http:     withoutRedirects(&http.Client{}),
		timeout:  jwksFetchTimeout,
		maxStale: jwksDefaultMaxStale,
		now:      time.Now,
	}
	for _, opt := range opts {
		opt(c)
	}
	if !c.acceptsURL(rawURL) {
		return nil, ErrJWKSURLInvalid
	}
	if c.maxStale <= 0 {
		return nil, ErrMaxStaleInvalid
	}
	return c, nil
}

func (c *JWKSClient) acceptsURL(rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil || u.Host == "" || u.User != nil {
		return false
	}
	return u.Scheme == "https" || (u.Scheme == "http" && c.allowLoopback && isLoopback(u.Hostname()))
}

func isLoopback(host string) bool {
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func withoutRedirects(hc *http.Client) *http.Client {
	clone := *hc
	clone.CheckRedirect = func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}
	return &clone
}

// Lookup implements KeySet. A cached unexpired key is returned without
// blocking. Otherwise Lookup may wait for a refresh until ctx ends, then falls
// back to the key it already holds if that key is within the WithMaxStale limit.
func (c *JWKSClient) Lookup(ctx context.Context, kid string) (Key, error) {
	if key, ok := c.freshKey(kid); ok {
		return key, nil
	}
	waitErr := c.refresh(ctx)

	c.mu.Lock()
	defer c.mu.Unlock()
	if c.keys == nil || c.now().Sub(c.lastSuccess) > c.maxStale {
		return Key{}, c.unavailable(waitErr)
	}
	if key, ok := c.keys.get(kid); ok {
		return key, nil
	}
	if waitErr != nil {
		return Key{}, c.unavailable(waitErr)
	}
	return Key{}, ErrKeyNotFound
}

// unavailable wraps the reason no key can be served. There is no reason to
// report when the keys outlived a limit shorter than the refresh interval.
func (c *JWKSClient) unavailable(waitErr error) error {
	cause := waitErr
	if cause == nil {
		cause = c.lastErr
	}
	if cause == nil {
		return ErrKeySetUnavailable
	}
	return fmt.Errorf("%w: %w", ErrKeySetUnavailable, cause)
}

func (c *JWKSClient) freshKey(kid string) (Key, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.now().Before(c.expires) {
		return Key{}, false
	}
	return c.keys.get(kid)
}

// refresh joins a running fetch or, when the rate limit allows, starts one and
// waits for it. The fetch outlives ctx's cancellation because other callers may
// be waiting on it, and it has its own timeout. A non-nil result is ctx's error.
// Fetch failures are recorded in lastErr.
func (c *JWKSClient) refresh(ctx context.Context) error {
	c.mu.Lock()
	done := c.inflight
	if done == nil {
		now := c.now()
		if !c.lastAttempt.IsZero() && now.Sub(c.lastAttempt) < jwksMinRefreshInterval {
			c.mu.Unlock()
			return nil
		}
		c.lastAttempt = now
		done = make(chan struct{})
		c.inflight = done
		go c.runFetch(context.WithoutCancel(ctx), done, now)
	}
	c.mu.Unlock()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (c *JWKSClient) runFetch(ctx context.Context, done chan struct{}, started time.Time) {
	defer close(done)
	keys, ttl, err := c.fetch(ctx)

	c.mu.Lock()
	defer c.mu.Unlock()
	c.inflight = nil
	c.lastErr = err
	if err == nil {
		c.keys = keys
		c.lastSuccess = started
		// A cached key must not outlive the limit Lookup enforces.
		c.expires = started.Add(min(ttl, c.maxStale))
	}
}

func (c *JWKSClient) fetch(ctx context.Context) (*StaticKeySet, time.Duration, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.url, nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, 0, fmt.Errorf("%w: status %d", ErrJWKSInvalid, resp.StatusCode)
	}
	// The limit applies after transparent gzip decoding.
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxJWKSBytes+1))
	if err != nil {
		return nil, 0, fmt.Errorf("%w: %w", ErrJWKSInvalid, err)
	}
	keys, err := ParseKeySet(body)
	if err != nil {
		return nil, 0, err
	}
	return keys, cacheTTL(resp.Header), nil
}

// cacheTTL returns the shortest valid max-age below the default TTL.
func cacheTTL(h http.Header) time.Duration {
	ttl := jwksDefaultTTL
	for _, value := range h.Values("Cache-Control") {
		for _, directive := range strings.Split(value, ",") {
			name, arg, _ := strings.Cut(strings.TrimSpace(directive), "=")
			if !strings.EqualFold(name, "max-age") {
				continue
			}
			seconds, err := strconv.Atoi(arg)
			if err != nil || seconds < 0 || seconds >= int(ttl/time.Second) {
				continue
			}
			ttl = time.Duration(seconds) * time.Second
		}
	}
	return ttl
}
