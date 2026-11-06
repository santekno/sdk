package jwtx_test

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/santekno/sdk/jwtx"
)

// fakeClock is a settable clock safe for concurrent use.
type fakeClock struct {
	nanos atomic.Int64
}

func newFakeClock() *fakeClock {
	c := &fakeClock{}
	c.nanos.Store(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC).UnixNano())
	return c
}

func (c *fakeClock) now() time.Time          { return time.Unix(0, c.nanos.Load()) }
func (c *fakeClock) advance(d time.Duration) { c.nanos.Add(int64(d)) }

// jwksServer is an HTTPS JWKS endpoint whose response can change between
// requests. It counts requests.
type jwksServer struct {
	*httptest.Server

	hits atomic.Int64

	mu      sync.Mutex
	handler http.HandlerFunc
}

func newJWKSServer(t *testing.T, doc []byte) *jwksServer {
	t.Helper()
	s := &jwksServer{}
	s.serve(doc)
	s.Server = httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.hits.Add(1)
		s.mu.Lock()
		h := s.handler
		s.mu.Unlock()
		h(w, r)
	}))
	s.Config.ErrorLog = discardLog()
	// net/http retries a GET once when a reused connection drops, which would
	// show up as a second request in the counts these tests assert.
	s.Config.SetKeepAlivesEnabled(false)
	s.StartTLS()
	t.Cleanup(s.Close)
	return s
}

func (s *jwksServer) set(h http.HandlerFunc) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.handler = h
}

func (s *jwksServer) serve(doc []byte) {
	s.set(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write(doc) })
}

func (s *jwksServer) fail(status int) {
	s.set(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(status) })
}

func newClient(t *testing.T, s *jwksServer) (*jwtx.JWKSClient, *fakeClock) {
	t.Helper()
	c, err := jwtx.NewJWKSClient(s.URL, jwtx.WithHTTPClient(s.Client()))
	if err != nil {
		t.Fatalf("NewJWKSClient: %v", err)
	}
	clock := newFakeClock()
	c.SetClock(clock.now)
	return c, clock
}

func mustLookup(t *testing.T, c *jwtx.JWKSClient, kid string) {
	t.Helper()
	if _, err := c.Lookup(t.Context(), kid); err != nil {
		t.Fatalf("Lookup(%s): %v", kid, err)
	}
}

func discardLog() *log.Logger { return log.New(io.Discard, "", 0) }

func newECKey(t *testing.T) *ecdsa.PrivateKey {
	t.Helper()
	k, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	return k
}

func TestNewJWKSClient_URL(t *testing.T) {
	tests := []struct {
		name string
		url  string
		opts []jwtx.JWKSOption
		ok   bool
	}{
		{"https", "https://issuer.test/.well-known/jwks.json", nil, true},
		{"upper-case scheme", "HTTPS://issuer.test/jwks", nil, true},
		{"http", "http://issuer.test/jwks", nil, false},
		{"http loopback without the option", "http://127.0.0.1:54321/jwks", nil, false},
		{"ftp", "ftp://issuer.test/jwks", nil, false},
		{"file", "file:///etc/passwd", nil, false},
		{"no scheme", "issuer.test/jwks", nil, false},
		{"no host", "https:///jwks", nil, false},
		{"empty", "", nil, false},
		{"unparsable", "https://bad host/\x7f", nil, false},
		{"credentials", "https://user:pass@issuer.test/jwks", nil, false},
		{"loopback option on IPv4", "http://127.0.0.1:54321/jwks", []jwtx.JWKSOption{jwtx.WithInsecureLoopback()}, true},
		{"loopback option on IPv6", "http://[::1]:54321/jwks", []jwtx.JWKSOption{jwtx.WithInsecureLoopback()}, true},
		{"loopback option on localhost", "http://localhost:54321/jwks", []jwtx.JWKSOption{jwtx.WithInsecureLoopback()}, true},
		{"loopback option on a public host", "http://issuer.test/jwks", []jwtx.JWKSOption{jwtx.WithInsecureLoopback()}, false},
		{"loopback option on a public IP", "http://8.8.8.8/jwks", []jwtx.JWKSOption{jwtx.WithInsecureLoopback()}, false},
		{"loopback option on a lookalike host", "http://localhost.attacker.test/jwks", []jwtx.JWKSOption{jwtx.WithInsecureLoopback()}, false},
		{"loopback option with credentials", "http://u:p@127.0.0.1/jwks", []jwtx.JWKSOption{jwtx.WithInsecureLoopback()}, false},
		{"loopback option keeps https", "https://issuer.test/jwks", []jwtx.JWKSOption{jwtx.WithInsecureLoopback()}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := jwtx.NewJWKSClient(tt.url, tt.opts...)
			if tt.ok && (err != nil || c == nil) {
				t.Fatalf("got %v", err)
			}
			if !tt.ok && !errors.Is(err, jwtx.ErrJWKSURLInvalid) {
				t.Fatalf("got %v", err)
			}
		})
	}
}

func TestJWKSClient_LoopbackFetch(t *testing.T) {
	doc := jwksDoc(t, ecJWK(t, &ecKey(t).PublicKey, testKID))
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write(doc) }))
	defer srv.Close()

	c, err := jwtx.NewJWKSClient(srv.URL, jwtx.WithInsecureLoopback())
	if err != nil {
		t.Fatalf("NewJWKSClient: %v", err)
	}
	mustLookup(t, c, testKID)
}

func TestJWKSClient_NoRequestBeforeFirstLookup(t *testing.T) {
	srv := newJWKSServer(t, jwksDoc(t, ecJWK(t, &ecKey(t).PublicKey, testKID)))
	newClient(t, srv)
	if got := srv.hits.Load(); got != 0 {
		t.Errorf("hits = %d", got)
	}
}

func TestJWKSClient_VerifiesTokens(t *testing.T) {
	ecPriv, rsaPriv := ecKey(t), rsaKey(t)
	srv := newJWKSServer(t, jwksDoc(t, ecJWK(t, &ecPriv.PublicKey, "ec"), rsaJWK(&rsaPriv.PublicKey, "rs")))
	c, _ := newClient(t, srv)

	tokens := []string{
		signES256(t, ecPriv, es256Header("ec"), validClaims()),
		signRS256(t, rsaPriv, map[string]any{"alg": "RS256", "kid": "rs"}, validClaims()),
	}
	for _, tok := range tokens {
		claims, err := jwtx.VerifyWithKeySet(tok, c, allowAsymmetric,
			jwtx.WithLeeway(30*time.Second), jwtx.WithContext(t.Context()))
		if err != nil {
			t.Fatalf("VerifyWithKeySet: %v", err)
		}
		if claims.Subject != "user-1" {
			t.Errorf("Subject = %q", claims.Subject)
		}
	}
	if got := srv.hits.Load(); got != 1 {
		t.Errorf("hits = %d, want 1", got)
	}
}

func TestJWKSClient_CacheLifetime(t *testing.T) {
	doc := jwksDoc(t, ecJWK(t, &ecKey(t).PublicKey, testKID))
	tests := []struct {
		name         string
		cacheControl []string
		want         time.Duration
	}{
		{"no header", nil, 10 * time.Minute},
		{"longer max-age is capped", []string{"public, max-age=86400"}, 10 * time.Minute},
		{"equal max-age", []string{"max-age=600"}, 10 * time.Minute},
		{"shorter max-age", []string{"public, max-age=120"}, 2 * time.Minute},
		{"upper-case directive", []string{"Max-Age=120"}, 2 * time.Minute},
		{"spaces around the directive", []string{"public ,  max-age=120 , must-revalidate"}, 2 * time.Minute},
		{"second header line", []string{"public", "max-age=120"}, 2 * time.Minute},
		{"shortest of two", []string{"max-age=300, max-age=120"}, 2 * time.Minute},
		{"s-maxage is not max-age", []string{"s-maxage=120"}, 10 * time.Minute},
		{"directive without a value", []string{"max-age"}, 10 * time.Minute},
		{"not a number", []string{"max-age=soon"}, 10 * time.Minute},
		{"negative", []string{"max-age=-5"}, 10 * time.Minute},
		{"overflow", []string{"max-age=99999999999999999999"}, 10 * time.Minute},
		{"huge but parsable", []string{"max-age=9223372036854775807"}, 10 * time.Minute},
		{"zero is floored by the refresh limit", []string{"max-age=0"}, 30 * time.Second},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := newJWKSServer(t, doc)
			srv.set(func(w http.ResponseWriter, _ *http.Request) {
				for _, v := range tt.cacheControl {
					w.Header().Add("Cache-Control", v)
				}
				_, _ = w.Write(doc)
			})
			c, clock := newClient(t, srv)

			mustLookup(t, c, testKID)
			clock.advance(tt.want - time.Second)
			mustLookup(t, c, testKID)
			if got := srv.hits.Load(); got != 1 {
				t.Fatalf("before expiry: hits = %d, want 1", got)
			}
			clock.advance(time.Second)
			mustLookup(t, c, testKID)
			if got := srv.hits.Load(); got != 2 {
				t.Fatalf("at expiry: hits = %d, want 2", got)
			}
		})
	}
}

// An attacker can pick any "kid". A cache miss must never translate into more
// than one upstream request per 30 seconds, however many tokens arrive.
func TestJWKSClient_RandomKidFloodIsRateLimited(t *testing.T) {
	tests := []struct {
		name  string
		setup func(*jwksServer)
	}{
		{"healthy upstream", func(*jwksServer) {}},
		{"failing upstream", func(s *jwksServer) { s.fail(http.StatusServiceUnavailable) }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := newJWKSServer(t, jwksDoc(t, ecJWK(t, &ecKey(t).PublicKey, testKID)))
			c, clock := newClient(t, srv)
			mustLookup(t, c, testKID)
			tt.setup(srv)

			flood := func(prefix string) {
				var wg sync.WaitGroup
				for g := range 8 {
					wg.Go(func() {
						for i := range 250 {
							_, err := c.Lookup(t.Context(), fmt.Sprintf("%s-%d-%d", prefix, g, i))
							if !errors.Is(err, jwtx.ErrKeyNotFound) {
								t.Errorf("Lookup: got %v", err)
							}
						}
					})
				}
				wg.Wait()
			}

			flood("a")
			if got := srv.hits.Load(); got != 1 {
				t.Fatalf("hits = %d, want 1: the initial load already spent the window", got)
			}
			clock.advance(29 * time.Second)
			flood("b")
			if got := srv.hits.Load(); got != 1 {
				t.Fatalf("after 29s: hits = %d, want 1", got)
			}
			clock.advance(time.Second)
			flood("c")
			if got := srv.hits.Load(); got != 2 {
				t.Fatalf("after 30s: hits = %d, want 2", got)
			}
		})
	}
}

func TestJWKSClient_SingleFlight(t *testing.T) {
	doc := jwksDoc(t, ecJWK(t, &ecKey(t).PublicKey, testKID))
	srv := newJWKSServer(t, doc)
	release := make(chan struct{})
	srv.set(func(w http.ResponseWriter, _ *http.Request) {
		<-release
		_, _ = w.Write(doc)
	})
	c, _ := newClient(t, srv)

	const callers = 50
	errs := make(chan error, callers)
	for range callers {
		go func() {
			_, err := c.Lookup(context.Background(), testKID)
			errs <- err
		}()
	}
	for srv.hits.Load() == 0 {
		time.Sleep(time.Millisecond)
	}
	time.Sleep(20 * time.Millisecond) // let the other callers queue up
	close(release)
	for range callers {
		if err := <-errs; err != nil {
			t.Errorf("Lookup: %v", err)
		}
	}
	if got := srv.hits.Load(); got != 1 {
		t.Errorf("hits = %d, want 1", got)
	}
}

func TestJWKSClient_RotationPicksUpNewKid(t *testing.T) {
	first, second := ecKey(t), newECKey(t)
	srv := newJWKSServer(t, jwksDoc(t, ecJWK(t, &first.PublicKey, "old")))
	c, clock := newClient(t, srv)
	mustLookup(t, c, "old")

	srv.serve(jwksDoc(t, ecJWK(t, &first.PublicKey, "old"), ecJWK(t, &second.PublicKey, "new")))
	if _, err := c.Lookup(t.Context(), "new"); !errors.Is(err, jwtx.ErrKeyNotFound) {
		t.Fatalf("inside the window: got %v", err)
	}
	clock.advance(30 * time.Second)
	mustLookup(t, c, "new")
	if got := srv.hits.Load(); got != 2 {
		t.Errorf("hits = %d, want 2", got)
	}
}

// Removing a key from the JWKS is how an issuer revokes it. A successful
// refresh must drop the key instead of keeping it until a per-key expiry.
func TestJWKSClient_RemovedKeyStopsWorking(t *testing.T) {
	first, second := ecKey(t), newECKey(t)
	srv := newJWKSServer(t, jwksDoc(t, ecJWK(t, &first.PublicKey, "old"), ecJWK(t, &second.PublicKey, "new")))
	c, clock := newClient(t, srv)
	mustLookup(t, c, "old")

	srv.serve(jwksDoc(t, ecJWK(t, &second.PublicKey, "new")))
	clock.advance(10 * time.Minute)
	if _, err := c.Lookup(t.Context(), "old"); !errors.Is(err, jwtx.ErrKeyNotFound) {
		t.Errorf("got %v", err)
	}
	mustLookup(t, c, "new")
}

func TestJWKSClient_KeepsOldKeysWhenRefreshFails(t *testing.T) {
	doc := jwksDoc(t, ecJWK(t, &ecKey(t).PublicKey, testKID))
	big := bytes.Repeat([]byte(" "), 64<<10+1)
	body := func(status int, data []byte) http.HandlerFunc {
		return func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(status)
			_, _ = w.Write(data)
		}
	}
	tests := []struct {
		name    string
		handler http.HandlerFunc
	}{
		{"status 500", body(http.StatusInternalServerError, doc)},
		{"status 404", body(http.StatusNotFound, nil)},
		{"status 204", body(http.StatusNoContent, nil)},
		{"not JSON", body(http.StatusOK, []byte("<html>maintenance</html>"))},
		{"no usable keys", body(http.StatusOK, []byte(`{"keys":[]}`))},
		{"too large", body(http.StatusOK, big)},
		{"connection dropped", func(w http.ResponseWriter, _ *http.Request) {
			panic(http.ErrAbortHandler)
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := newJWKSServer(t, doc)
			c, clock := newClient(t, srv)
			mustLookup(t, c, testKID)

			srv.set(tt.handler)
			clock.advance(10 * time.Minute)
			mustLookup(t, c, testKID)
			if got := srv.hits.Load(); got != 2 {
				t.Fatalf("hits = %d, want 2", got)
			}

			clock.advance(29 * time.Second)
			mustLookup(t, c, testKID)
			if got := srv.hits.Load(); got != 2 {
				t.Fatalf("failing upstream hit inside the window: hits = %d", got)
			}

			srv.serve(doc)
			clock.advance(time.Second)
			mustLookup(t, c, testKID)
			clock.advance(time.Second)
			mustLookup(t, c, testKID)
			if got := srv.hits.Load(); got != 3 {
				t.Fatalf("after recovery: hits = %d, want 3", got)
			}
		})
	}
}

func TestJWKSClient_NoKeysEverLoaded(t *testing.T) {
	doc := jwksDoc(t, ecJWK(t, &ecKey(t).PublicKey, testKID))
	srv := newJWKSServer(t, doc)
	srv.fail(http.StatusBadGateway)
	c, clock := newClient(t, srv)

	tests := []struct {
		name     string
		wantHits int64
	}{
		{"first attempt", 1},
		{"second attempt inside the window", 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tok := signES256(t, ecKey(t), es256Header(testKID), validClaims())
			_, err := jwtx.VerifyWithKeySet(tok, c, allowAsymmetric)
			if !errors.Is(err, jwtx.ErrKeySetUnavailable) {
				t.Fatalf("got %v", err)
			}
			if got := srv.hits.Load(); got != tt.wantHits {
				t.Errorf("hits = %d, want %d", got, tt.wantHits)
			}
		})
	}

	srv.serve(doc)
	clock.advance(30 * time.Second)
	mustLookup(t, c, testKID)
}

func TestNewJWKSClient_MaxStale(t *testing.T) {
	const url = "https://issuer.test/jwks"
	tests := []struct {
		name     string
		maxStale time.Duration
		want     error
	}{
		{"zero", 0, jwtx.ErrMaxStaleInvalid},
		{"negative", -time.Nanosecond, jwtx.ErrMaxStaleInvalid},
		{"large negative", -24 * time.Hour, jwtx.ErrMaxStaleInvalid},
		{"smallest positive", time.Nanosecond, nil},
		{"longer than the default", 7 * 24 * time.Hour, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := jwtx.NewJWKSClient(url, jwtx.WithMaxStale(tt.maxStale))
			if !errors.Is(err, tt.want) {
				t.Fatalf("got %v, want %v", err, tt.want)
			}
			if (c == nil) != (tt.want != nil) {
				t.Errorf("client = %v with error %v", c, err)
			}
		})
	}
}

// newStaleClient returns a client whose keys loaded at the fake clock's start
// and a server that fails every later request.
func newStaleClient(t *testing.T, opts ...jwtx.JWKSOption) (*jwtx.JWKSClient, *fakeClock, *jwksServer, []byte) {
	t.Helper()
	doc := jwksDoc(t, ecJWK(t, &ecKey(t).PublicKey, testKID))
	srv := newJWKSServer(t, doc)
	c, err := jwtx.NewJWKSClient(srv.URL, append([]jwtx.JWKSOption{jwtx.WithHTTPClient(srv.Client())}, opts...)...)
	if err != nil {
		t.Fatalf("NewJWKSClient: %v", err)
	}
	clock := newFakeClock()
	c.SetClock(clock.now)
	mustLookup(t, c, testKID)
	srv.fail(http.StatusServiceUnavailable)
	return c, clock, srv, doc
}

func wantUnavailable(t *testing.T, c *jwtx.JWKSClient, kid string) {
	t.Helper()
	if _, err := c.Lookup(t.Context(), kid); !errors.Is(err, jwtx.ErrKeySetUnavailable) {
		t.Fatalf("Lookup(%s): got %v, want ErrKeySetUnavailable", kid, err)
	}
}

func TestJWKSClient_MaxStaleBoundary(t *testing.T) {
	tests := []struct {
		name  string
		opts  []jwtx.JWKSOption
		limit time.Duration
	}{
		{"default of 24 hours", nil, 24 * time.Hour},
		{"explicit 24 hours", []jwtx.JWKSOption{jwtx.WithMaxStale(24 * time.Hour)}, 24 * time.Hour},
		{"shorter limit", []jwtx.JWKSOption{jwtx.WithMaxStale(time.Hour)}, time.Hour},
		{"longer limit", []jwtx.JWKSOption{jwtx.WithMaxStale(48 * time.Hour)}, 48 * time.Hour},
		{"limit below the cache lifetime", []jwtx.JWKSOption{jwtx.WithMaxStale(2 * time.Minute)}, 2 * time.Minute},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, clock, srv, _ := newStaleClient(t, tt.opts...)

			clock.advance(tt.limit)
			mustLookup(t, c, testKID) // exactly at the limit
			hits := srv.hits.Load()

			clock.advance(time.Nanosecond)
			wantUnavailable(t, c, testKID)
			wantUnavailable(t, c, "unknown-kid")
			if got := srv.hits.Load(); got != hits {
				t.Errorf("hits = %d, want %d: still inside the refresh window", got, hits)
			}
		})
	}
}

func TestJWKSClient_MaxStaleErrorKeepsCause(t *testing.T) {
	c, clock, _, _ := newStaleClient(t)
	clock.advance(24*time.Hour + time.Second)

	_, err := c.Lookup(t.Context(), "SECRET-KID")
	if !errors.Is(err, jwtx.ErrKeySetUnavailable) || !errors.Is(err, jwtx.ErrJWKSInvalid) {
		t.Fatalf("got %v", err)
	}
	if strings.Contains(err.Error(), "SECRET-KID") {
		t.Errorf("error echoes the kid: %q", err)
	}
}

// The age counts from the start of the fetch: the keys are as old as the
// moment the issuer was asked, however slowly it answered.
func TestJWKSClient_MaxStaleCountsFromFetchStart(t *testing.T) {
	doc := jwksDoc(t, ecJWK(t, &ecKey(t).PublicKey, testKID))
	srv := newJWKSServer(t, doc)
	c, clock := newClient(t, srv)
	srv.set(func(w http.ResponseWriter, _ *http.Request) {
		clock.advance(time.Minute)
		_, _ = w.Write(doc)
	})
	mustLookup(t, c, testKID)
	srv.fail(http.StatusServiceUnavailable)

	clock.advance(24*time.Hour - time.Minute)
	mustLookup(t, c, testKID)
	clock.advance(time.Nanosecond)
	wantUnavailable(t, c, testKID)
}

func TestJWKSClient_TokensVerifyWhileRefreshFailsBelowLimit(t *testing.T) {
	c, clock, srv, _ := newStaleClient(t)
	tok := signES256(t, ecKey(t), es256Header(testKID), validClaims())

	for hour := 1; hour <= 23; hour++ {
		clock.advance(time.Hour)
		claims, err := jwtx.VerifyWithKeySet(tok, c, allowAsymmetric, jwtx.WithContext(t.Context()))
		if err != nil {
			t.Fatalf("after %d hours: %v", hour, err)
		}
		if claims.Subject != "user-1" {
			t.Errorf("Subject = %q", claims.Subject)
		}
	}
	if got, want := srv.hits.Load(), int64(1+23); got != want {
		t.Errorf("hits = %d, want %d: one failed refresh per hour", got, want)
	}

	clock.advance(time.Hour + time.Nanosecond)
	if _, err := jwtx.VerifyWithKeySet(tok, c, allowAsymmetric); !errors.Is(err, jwtx.ErrKeySetUnavailable) {
		t.Errorf("past the limit: got %v", err)
	}
}

func TestJWKSClient_MaxStaleRecovery(t *testing.T) {
	c, clock, srv, doc := newStaleClient(t)

	clock.advance(24*time.Hour + time.Second)
	wantUnavailable(t, c, testKID)

	srv.serve(doc)
	wantUnavailable(t, c, testKID) // recovery waits for the refresh window
	clock.advance(30 * time.Second)
	mustLookup(t, c, testKID)

	// The limit restarts from the successful fetch.
	recovered := srv.hits.Load()
	srv.fail(http.StatusServiceUnavailable)
	clock.advance(24 * time.Hour)
	mustLookup(t, c, testKID)
	clock.advance(time.Nanosecond)
	wantUnavailable(t, c, testKID)
	if got := srv.hits.Load(); got != recovered+1 {
		t.Errorf("hits = %d, want %d", got, recovered+1)
	}
}

// Failed attempts must not count as freshness, otherwise a blocked endpoint
// would extend the life of the keys forever.
func TestJWKSClient_FailedRefreshDoesNotRestartLimit(t *testing.T) {
	c, clock, srv, _ := newStaleClient(t)

	clock.advance(23*time.Hour + 59*time.Minute)
	mustLookup(t, c, testKID)
	hits := srv.hits.Load()

	clock.advance(time.Minute + time.Nanosecond)
	wantUnavailable(t, c, testKID)
	if got := srv.hits.Load(); got != hits+1 {
		t.Errorf("hits = %d, want a refresh attempt past the limit", got)
	}
}

func TestJWKSClient_IdleClientRecoversOnFirstLookup(t *testing.T) {
	c, clock, srv, doc := newStaleClient(t)
	srv.serve(doc)

	clock.advance(7 * 24 * time.Hour)
	mustLookup(t, c, testKID)
	if got := srv.hits.Load(); got != 2 {
		t.Errorf("hits = %d, want 2", got)
	}
}

// A limit below the cache lifetime must force a refresh when it is reached.
// A healthy issuer then sees no gap in service.
func TestJWKSClient_MaxStaleShortensCacheLifetime(t *testing.T) {
	doc := jwksDoc(t, ecJWK(t, &ecKey(t).PublicKey, testKID))
	srv := newJWKSServer(t, doc)
	c, err := jwtx.NewJWKSClient(srv.URL, jwtx.WithHTTPClient(srv.Client()), jwtx.WithMaxStale(2*time.Minute))
	if err != nil {
		t.Fatalf("NewJWKSClient: %v", err)
	}
	clock := newFakeClock()
	c.SetClock(clock.now)

	mustLookup(t, c, testKID)
	clock.advance(2*time.Minute - time.Second)
	mustLookup(t, c, testKID)
	if got := srv.hits.Load(); got != 1 {
		t.Fatalf("before the limit: hits = %d, want 1", got)
	}
	clock.advance(time.Second)
	mustLookup(t, c, testKID)
	if got := srv.hits.Load(); got != 2 {
		t.Fatalf("at the limit: hits = %d, want 2", got)
	}
}

func TestJWKSClient_MaxStaleFloodIsRateLimited(t *testing.T) {
	c, clock, srv, _ := newStaleClient(t)
	clock.advance(24*time.Hour + time.Second)
	wantUnavailable(t, c, testKID)
	hits := srv.hits.Load()

	var wg sync.WaitGroup
	for g := range 8 {
		wg.Go(func() {
			for i := range 250 {
				_, err := c.Lookup(t.Context(), fmt.Sprintf("kid-%d-%d", g, i))
				if !errors.Is(err, jwtx.ErrKeySetUnavailable) {
					t.Errorf("Lookup: got %v", err)
				}
			}
		})
	}
	wg.Wait()
	if got := srv.hits.Load(); got != hits {
		t.Fatalf("hits = %d, want %d", got, hits)
	}
	clock.advance(30 * time.Second)
	wantUnavailable(t, c, "another-kid")
	if got := srv.hits.Load(); got != hits+1 {
		t.Errorf("after 30s: hits = %d, want %d", got, hits+1)
	}
}

func TestJWKSClient_MaxStaleWithCanceledCaller(t *testing.T) {
	c, clock, srv, _ := newStaleClient(t)
	release := make(chan struct{})
	t.Cleanup(func() { close(release) })
	srv.set(func(http.ResponseWriter, *http.Request) { <-release })
	clock.advance(24*time.Hour + time.Second)

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err := c.Lookup(ctx, testKID)
	if !errors.Is(err, jwtx.ErrKeySetUnavailable) || !errors.Is(err, context.Canceled) {
		t.Errorf("got %v", err)
	}
}

// With a limit below the refresh interval no fetch has failed, so there is no
// cause to wrap. The error must still be well formed.
func TestJWKSClient_MaxStaleBelowRefreshInterval(t *testing.T) {
	doc := jwksDoc(t, ecJWK(t, &ecKey(t).PublicKey, testKID))
	srv := newJWKSServer(t, doc)
	c, err := jwtx.NewJWKSClient(srv.URL, jwtx.WithHTTPClient(srv.Client()), jwtx.WithMaxStale(10*time.Second))
	if err != nil {
		t.Fatalf("NewJWKSClient: %v", err)
	}
	clock := newFakeClock()
	c.SetClock(clock.now)
	mustLookup(t, c, testKID)

	clock.advance(11 * time.Second)
	_, err = c.Lookup(t.Context(), testKID)
	if !errors.Is(err, jwtx.ErrKeySetUnavailable) {
		t.Fatalf("got %v", err)
	}
	if strings.Contains(err.Error(), "%!") {
		t.Errorf("malformed error: %q", err)
	}
	if got := srv.hits.Load(); got != 1 {
		t.Errorf("hits = %d, want 1", got)
	}
}

func TestJWKSClient_RefusesRedirects(t *testing.T) {
	doc := jwksDoc(t, ecJWK(t, &ecKey(t).PublicKey, testKID))
	var plainHits, tlsHits atomic.Int64
	plain := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		plainHits.Add(1)
		_, _ = w.Write(doc)
	}))
	defer plain.Close()
	secure := newJWKSServer(t, doc)
	secure.set(func(w http.ResponseWriter, _ *http.Request) {
		tlsHits.Add(1)
		_, _ = w.Write(doc)
	})

	tests := []struct {
		name   string
		target string
		status int
	}{
		{"https to http", plain.URL, http.StatusFound},
		{"https to https", secure.URL, http.StatusMovedPermanently},
		{"permanent to http", plain.URL, http.StatusPermanentRedirect},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			origin := newJWKSServer(t, nil)
			origin.set(func(w http.ResponseWriter, r *http.Request) {
				http.Redirect(w, r, tt.target, tt.status)
			})
			permissive := *secure.Client()
			permissive.CheckRedirect = func(*http.Request, []*http.Request) error { return nil }
			c, err := jwtx.NewJWKSClient(origin.URL, jwtx.WithHTTPClient(&permissive))
			if err != nil {
				t.Fatalf("NewJWKSClient: %v", err)
			}
			if _, err := c.Lookup(t.Context(), testKID); !errors.Is(err, jwtx.ErrKeySetUnavailable) {
				t.Errorf("got %v", err)
			}
		})
	}
	if plainHits.Load() != 0 || tlsHits.Load() != 0 {
		t.Errorf("redirect followed: plain=%d tls=%d", plainHits.Load(), tlsHits.Load())
	}
}

func TestJWKSClient_RefusesOversizedBody(t *testing.T) {
	var gz bytes.Buffer
	w := gzip.NewWriter(&gz)
	if _, err := w.Write(bytes.Repeat([]byte(" "), 8<<20)); err != nil {
		t.Fatalf("gzip: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("gzip: %v", err)
	}
	tests := []struct {
		name    string
		handler http.HandlerFunc
	}{
		{"declared length", func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write(bytes.Repeat([]byte(" "), 1<<20))
		}},
		{"chunked", func(w http.ResponseWriter, _ *http.Request) {
			for range 128 {
				_, _ = w.Write(bytes.Repeat([]byte(" "), 1<<10))
				w.(http.Flusher).Flush()
			}
		}},
		{"small compressed body expanding past the limit", func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Encoding", "gzip")
			_, _ = w.Write(gz.Bytes())
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := newJWKSServer(t, nil)
			srv.set(tt.handler)
			c, _ := newClient(t, srv)
			_, err := c.Lookup(t.Context(), testKID)
			if !errors.Is(err, jwtx.ErrKeySetUnavailable) || !errors.Is(err, jwtx.ErrJWKSInvalid) {
				t.Errorf("got %v", err)
			}
		})
	}
}

// The read must stop at the limit rather than buffer until the timeout.
func TestJWKSClient_StopsReadingEndlessBody(t *testing.T) {
	srv := newJWKSServer(t, nil)
	srv.set(func(w http.ResponseWriter, r *http.Request) {
		chunk := bytes.Repeat([]byte(" "), 1<<10)
		for r.Context().Err() == nil {
			if _, err := w.Write(chunk); err != nil {
				return
			}
		}
	})
	c, _ := newClient(t, srv)
	c.SetTimeout(time.Second)

	_, err := c.Lookup(t.Context(), testKID)
	if !errors.Is(err, jwtx.ErrJWKSInvalid) || errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("got %v", err)
	}
}

func TestJWKSClient_Timeout(t *testing.T) {
	doc := jwksDoc(t, ecJWK(t, &ecKey(t).PublicKey, testKID))
	tests := []struct {
		name    string
		handler func(release <-chan struct{}) http.HandlerFunc
	}{
		{"stalled headers", func(release <-chan struct{}) http.HandlerFunc {
			return func(http.ResponseWriter, *http.Request) { <-release }
		}},
		{"stalled body", func(release <-chan struct{}) http.HandlerFunc {
			return func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write(doc[:10])
				w.(http.Flusher).Flush()
				<-release
			}
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := newJWKSServer(t, doc)
			release := make(chan struct{})
			t.Cleanup(func() { close(release) })
			c, _ := newClient(t, srv)
			c.SetTimeout(50 * time.Millisecond)
			srv.set(tt.handler(release))

			start := time.Now()
			_, err := c.Lookup(t.Context(), testKID)
			if !errors.Is(err, jwtx.ErrKeySetUnavailable) || !errors.Is(err, context.DeadlineExceeded) {
				t.Errorf("got %v", err)
			}
			if took := time.Since(start); took > 2*time.Second {
				t.Errorf("took %v", took)
			}
		})
	}
}

func TestJWKSClient_CallerContext(t *testing.T) {
	doc := jwksDoc(t, ecJWK(t, &ecKey(t).PublicKey, testKID))
	// stall makes srv hang until the end of the subtest. Registered after the
	// server, so it runs before the server's cleanup, which waits for handlers.
	stall := func(t *testing.T, srv *jwksServer) {
		t.Helper()
		release := make(chan struct{})
		t.Cleanup(func() { close(release) })
		srv.set(func(http.ResponseWriter, *http.Request) { <-release })
	}
	canceledContext := func() context.Context {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		return ctx
	}

	t.Run("no key held", func(t *testing.T) {
		srv := newJWKSServer(t, nil)
		stall(t, srv)
		c, _ := newClient(t, srv)
		tok := signES256(t, ecKey(t), es256Header(testKID), validClaims())
		_, err := jwtx.VerifyWithKeySet(tok, c, allowAsymmetric, jwtx.WithContext(canceledContext()))
		if !errors.Is(err, jwtx.ErrKeySetUnavailable) || !errors.Is(err, context.Canceled) {
			t.Errorf("got %v", err)
		}
	})

	t.Run("expired key held", func(t *testing.T) {
		srv := newJWKSServer(t, doc)
		c, clock := newClient(t, srv)
		mustLookup(t, c, testKID)
		stall(t, srv)
		clock.advance(10 * time.Minute)
		if _, err := c.Lookup(canceledContext(), testKID); err != nil {
			t.Errorf("expired key not served: %v", err)
		}
	})
}

// A caller that gives up must not cancel the fetch other callers wait on.
func TestJWKSClient_CanceledCallerDoesNotAbortFetch(t *testing.T) {
	doc := jwksDoc(t, ecJWK(t, &ecKey(t).PublicKey, testKID))
	srv := newJWKSServer(t, doc)
	release := make(chan struct{})
	srv.set(func(w http.ResponseWriter, _ *http.Request) {
		<-release
		_, _ = w.Write(doc)
	})
	c, _ := newClient(t, srv)

	ctx, cancel := context.WithCancel(t.Context())
	first := make(chan error, 1)
	go func() {
		_, err := c.Lookup(ctx, testKID)
		first <- err
	}()
	for srv.hits.Load() == 0 {
		time.Sleep(time.Millisecond)
	}
	second := make(chan error, 1)
	go func() {
		_, err := c.Lookup(context.Background(), testKID)
		second <- err
	}()
	cancel()
	if err := <-first; !errors.Is(err, context.Canceled) {
		t.Errorf("first caller: got %v", err)
	}
	close(release)
	if err := <-second; err != nil {
		t.Errorf("second caller: %v", err)
	}
	if got := srv.hits.Load(); got != 1 {
		t.Errorf("hits = %d, want 1", got)
	}
}

func TestWithHTTPClient_NilKeepsDefault(t *testing.T) {
	if _, err := jwtx.NewJWKSClient("https://issuer.test/jwks", jwtx.WithHTTPClient(nil)); err != nil {
		t.Errorf("got %v", err)
	}
}

func TestWithContext_NilKeepsDefault(t *testing.T) {
	priv := ecKey(t)
	ks := parseSet(t, jwksDoc(t, ecJWK(t, &priv.PublicKey, testKID)))
	tok := signES256(t, priv, es256Header(testKID), validClaims())
	//nolint:staticcheck // a nil context is the case under test
	if _, err := jwtx.VerifyWithKeySet(tok, ks, allowAsymmetric, jwtx.WithContext(nil)); err != nil {
		t.Errorf("got %v", err)
	}
}

func TestJWKSClient_ErrorsDoNotEchoBody(t *testing.T) {
	const marker = "SECRET-MARKER"
	tests := []struct {
		name   string
		status int
	}{
		{"error status", http.StatusInternalServerError},
		{"invalid document", http.StatusOK},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := newJWKSServer(t, nil)
			srv.set(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(marker))
			})
			c, _ := newClient(t, srv)
			_, err := c.Lookup(t.Context(), marker)
			if err == nil {
				t.Fatal("no error")
			}
			if strings.Contains(err.Error(), marker) {
				t.Errorf("error echoes the response or kid: %q", err)
			}
		})
	}
}
