package ratelimitx_test

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/santekno/sdk/ratelimitx"
)

// fakeClient is an in-memory test double for the ratelimitx.Client interface.
type fakeClient struct {
	mu       sync.Mutex
	counts   map[string]int64
	expires  map[string]time.Time
	now      func() time.Time
	failIncr error
	failPTTL error
	failExp  error
}

func newFakeClient() *fakeClient {
	return &fakeClient{
		counts:  map[string]int64{},
		expires: map[string]time.Time{},
		now:     time.Now,
	}
}

func (f *fakeClient) Incr(_ context.Context, key string) (int64, error) {
	if f.failIncr != nil {
		return 0, f.failIncr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if exp, ok := f.expires[key]; ok && f.now().After(exp) {
		// TTL expired — reset.
		delete(f.counts, key)
		delete(f.expires, key)
	}
	f.counts[key]++
	return f.counts[key], nil
}

func (f *fakeClient) Expire(_ context.Context, key string, ttl time.Duration) error {
	if f.failExp != nil {
		return f.failExp
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.expires[key] = f.now().Add(ttl)
	return nil
}

func (f *fakeClient) PTTL(_ context.Context, key string) (time.Duration, error) {
	if f.failPTTL != nil {
		return 0, f.failPTTL
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	exp, ok := f.expires[key]
	if !ok {
		return -1, nil
	}
	d := exp.Sub(f.now())
	if d < 0 {
		return -1, nil
	}
	return d, nil
}

// ============================================================================
// Allow tests
// ============================================================================

func TestAllow_HappyPath_UnderLimit(t *testing.T) {
	fc := newFakeClient()
	limiter := ratelimitx.New(fc,
		ratelimitx.WithLimit(3),
		ratelimitx.WithWindow(time.Hour),
		ratelimitx.WithPrefix("test"),
	)

	for i := 1; i <= 3; i++ {
		res, err := limiter.Allow(context.Background(), "ip-1")
		if err != nil {
			t.Fatalf("Allow #%d unexpected error: %v", i, err)
		}
		if !res.Allowed {
			t.Errorf("Allow #%d: expected Allowed=true, got false", i)
		}
		wantRemaining := 3 - i
		if res.Remaining != wantRemaining {
			t.Errorf("Allow #%d: Remaining = %d, want %d", i, res.Remaining, wantRemaining)
		}
	}
}

func TestAllow_OverLimit_Rejects(t *testing.T) {
	fc := newFakeClient()
	limiter := ratelimitx.New(fc,
		ratelimitx.WithLimit(2),
		ratelimitx.WithWindow(time.Hour),
	)

	for range 2 {
		if _, err := limiter.Allow(context.Background(), "ip-1"); err != nil {
			t.Fatal(err)
		}
	}

	res, err := limiter.Allow(context.Background(), "ip-1")
	if err != nil {
		t.Fatal(err)
	}
	if res.Allowed {
		t.Error("expected Allowed=false on 3rd call (limit=2)")
	}
	if res.Remaining != 0 {
		t.Errorf("Remaining = %d, want 0", res.Remaining)
	}
	if res.RetryAfter <= 0 {
		t.Errorf("RetryAfter = %v, want positive duration", res.RetryAfter)
	}
}

func TestAllow_SeparateIDs_TrackSeparately(t *testing.T) {
	fc := newFakeClient()
	limiter := ratelimitx.New(fc,
		ratelimitx.WithLimit(2),
		ratelimitx.WithWindow(time.Hour),
	)

	// ip-1 hits 2/2
	for range 2 {
		limiter.Allow(context.Background(), "ip-1")
	}
	// ip-2 should still have full quota
	res, _ := limiter.Allow(context.Background(), "ip-2")
	if !res.Allowed {
		t.Error("ip-2 should be allowed (separate bucket)")
	}
	if res.Remaining != 1 {
		t.Errorf("ip-2 Remaining = %d, want 1", res.Remaining)
	}
}

func TestAllow_WindowExpiry_ResetsCount(t *testing.T) {
	fc := newFakeClient()
	currentTime := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	fc.now = func() time.Time { return currentTime }

	limiter := ratelimitx.New(fc,
		ratelimitx.WithLimit(1),
		ratelimitx.WithWindow(time.Hour),
		ratelimitx.WithNow(func() time.Time { return currentTime }),
	)

	// First request — allowed
	res, _ := limiter.Allow(context.Background(), "ip-1")
	if !res.Allowed {
		t.Fatal("first request should be allowed")
	}

	// Second request — rejected (same window)
	res, _ = limiter.Allow(context.Background(), "ip-1")
	if res.Allowed {
		t.Fatal("second request in same window should be rejected")
	}

	// Advance clock past the window
	currentTime = currentTime.Add(2 * time.Hour)

	// Third request — should be allowed (new bucket + expired TTL)
	res, _ = limiter.Allow(context.Background(), "ip-1")
	if !res.Allowed {
		t.Error("request after window expiry should be allowed")
	}
}

func TestAllow_BucketFormat(t *testing.T) {
	fc := newFakeClient()
	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	limiter := ratelimitx.New(fc,
		ratelimitx.WithLimit(10),
		ratelimitx.WithWindow(24*time.Hour),
		ratelimitx.WithNow(func() time.Time { return now }),
	)

	res, _ := limiter.Allow(context.Background(), "ip-1")
	if !strings.Contains(res.Bucket, "20260601") {
		t.Errorf("Bucket = %q, want it to encode the date", res.Bucket)
	}
}

func TestAllow_IncrError_PropagatesError(t *testing.T) {
	fc := newFakeClient()
	fc.failIncr = errors.New("redis down")

	limiter := ratelimitx.New(fc, ratelimitx.WithLimit(10))

	_, err := limiter.Allow(context.Background(), "ip-1")
	if err == nil {
		t.Error("expected error when Incr fails")
	}
	if !strings.Contains(err.Error(), "redis down") {
		t.Errorf("error should wrap upstream message, got: %v", err)
	}
}

func TestAllow_ExpireError_DoesNotFailRequest(t *testing.T) {
	fc := newFakeClient()
	fc.failExp = errors.New("expire failed")

	limiter := ratelimitx.New(fc, ratelimitx.WithLimit(10))

	res, err := limiter.Allow(context.Background(), "ip-1")
	if err != nil {
		t.Fatalf("Allow should succeed even when Expire fails: %v", err)
	}
	if !res.Allowed {
		t.Error("expected Allowed=true")
	}
}

func TestAllow_PTTLError_FallsBackToWindow(t *testing.T) {
	fc := newFakeClient()
	limiter := ratelimitx.New(fc,
		ratelimitx.WithLimit(1),
		ratelimitx.WithWindow(time.Hour),
	)

	// First request allowed
	limiter.Allow(context.Background(), "ip-1")

	// Now simulate PTTL failure on the rejection path
	fc.failPTTL = errors.New("pttl fail")

	res, err := limiter.Allow(context.Background(), "ip-1")
	if err != nil {
		t.Fatal(err)
	}
	if res.Allowed {
		t.Error("expected Allowed=false")
	}
	if res.RetryAfter != time.Hour {
		t.Errorf("RetryAfter = %v, want full window fallback (1h)", res.RetryAfter)
	}
}

// ============================================================================
// New / Option tests
// ============================================================================

func TestNew_NilClient_Panics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic on nil client")
		}
	}()
	ratelimitx.New(nil)
}

func TestNew_ZeroLimit_Panics(t *testing.T) {
	fc := newFakeClient()
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic on zero limit")
		}
	}()
	ratelimitx.New(fc, ratelimitx.WithLimit(0))
}

func TestNew_NegativeWindow_Panics(t *testing.T) {
	fc := newFakeClient()
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic on negative window")
		}
	}()
	ratelimitx.New(fc, ratelimitx.WithWindow(-time.Hour))
}

func TestNew_Defaults(t *testing.T) {
	fc := newFakeClient()
	// Should not panic; defaults are valid.
	limiter := ratelimitx.New(fc)

	// Verify defaults via behavior: should allow 10 requests in default 24h window.
	for i := range 10 {
		res, _ := limiter.Allow(context.Background(), "ip-default")
		if !res.Allowed {
			t.Fatalf("default config should allow 10 requests; failed at %d", i)
		}
	}
	res, _ := limiter.Allow(context.Background(), "ip-default")
	if res.Allowed {
		t.Error("11th request should be rejected on default limit=10")
	}
}
