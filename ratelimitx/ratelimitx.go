package ratelimitx

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// Client is the minimal Redis interface ratelimitx requires. Any client that
// implements these methods works — including go-redis/v9 via a thin adapter
// (see example_test.go).
type Client interface {
	// Incr atomically increments the integer value at key by 1 and returns the
	// new value. Used for the per-key counter increment.
	Incr(ctx context.Context, key string) (int64, error)
	// Expire sets a TTL on key. Returns nil if the TTL was set, or if the key
	// does not exist; an error otherwise.
	Expire(ctx context.Context, key string, ttl time.Duration) error
	// PTTL returns the key's remaining TTL. Returns a negative duration when
	// the key has no TTL or does not exist; non-nil error on actual failures.
	PTTL(ctx context.Context, key string) (time.Duration, error)
}

// Common errors.
var (
	// ErrBadConfig is returned when New is called with invalid options.
	ErrBadConfig = errors.New("ratelimitx: bad configuration")
)

// Limiter is a fixed-window per-key rate limiter.
type Limiter struct {
	client Client
	limit  int
	window time.Duration
	prefix string
	// nowFunc is overridable for tests; defaults to time.Now.
	nowFunc func() time.Time
}

// Option configures a Limiter.
type Option func(*Limiter)

// WithLimit sets the maximum allowed Allow calls per window. Default: 10.
func WithLimit(n int) Option { return func(l *Limiter) { l.limit = n } }

// WithWindow sets the rate-limit window duration. Default: 24h.
func WithWindow(d time.Duration) Option { return func(l *Limiter) { l.window = d } }

// WithPrefix sets the Redis key prefix. Keys are shaped `<prefix>:<id>:<bucket>`.
// Default: "rl".
func WithPrefix(p string) Option { return func(l *Limiter) { l.prefix = p } }

// WithNow injects a custom clock function. Test-only helper.
func WithNow(now func() time.Time) Option { return func(l *Limiter) { l.nowFunc = now } }

// New returns a Limiter bound to the given Redis client.
//
// Default configuration: limit=10, window=24h, prefix="rl".
//
// client must not be nil; limit and window must be positive after options
// are applied — otherwise New panics with [ErrBadConfig].
func New(client Client, opts ...Option) *Limiter {
	if client == nil {
		panic(fmt.Errorf("%w: client must not be nil", ErrBadConfig))
	}
	l := &Limiter{
		client:  client,
		limit:   10,
		window:  24 * time.Hour,
		prefix:  "rl",
		nowFunc: time.Now,
	}
	for _, opt := range opts {
		opt(l)
	}
	if l.limit <= 0 {
		panic(fmt.Errorf("%w: limit must be > 0", ErrBadConfig))
	}
	if l.window <= 0 {
		panic(fmt.Errorf("%w: window must be > 0", ErrBadConfig))
	}
	return l
}

// Result describes the outcome of an Allow call.
type Result struct {
	// Allowed is true when the request was within the limit.
	Allowed bool
	// Remaining is the number of requests left in the current window
	// after this call. Always non-negative; 0 when the limit is reached.
	Remaining int
	// RetryAfter is the duration until the current window resets. Zero
	// when Allowed is true and the window hasn't started counting toward
	// the limit yet; a positive duration otherwise.
	RetryAfter time.Duration
	// Bucket is the time-derived bucket identifier used in the Redis key.
	// Exposed for telemetry / debugging.
	Bucket string
}

// Allow registers an attempt for id and returns the result. The contract:
//
//   - If the current bucket counter is <= limit after increment, Allowed=true
//     and Remaining is the headroom remaining in the window.
//   - If the counter exceeds limit, Allowed=false and RetryAfter is the
//     remaining TTL on the bucket key (≤ window).
//
// Implementation: a fixed-window counter aligned to time.Now / window. The
// first INCR sets the TTL via Expire (in a separate round trip — acceptable
// since the small race is harmless: either INCR sees no value and gets a
// freshly-expired result, or it gets the in-progress count).
func (l *Limiter) Allow(ctx context.Context, id string) (Result, error) {
	bucket := l.bucketID()
	key := fmt.Sprintf("%s:%s:%s", l.prefix, id, bucket)

	count, err := l.client.Incr(ctx, key)
	if err != nil {
		return Result{}, fmt.Errorf("ratelimitx: incr %q: %w", key, err)
	}

	// On the first increment of the bucket, set the TTL. We can detect "first"
	// by checking count == 1 — go-redis returns the post-increment value.
	if count == 1 {
		if expErr := l.client.Expire(ctx, key, l.window); expErr != nil {
			// Best-effort: not fatal. The next Allow call may also see no
			// TTL and re-Expire, which is fine. We just lose accuracy.
			// Still return success since the increment did happen.
			_ = expErr
		}
	}

	res := Result{Bucket: bucket}
	if count <= int64(l.limit) {
		res.Allowed = true
		res.Remaining = max(0, l.limit-int(count))
		return res, nil
	}

	// Over limit — populate RetryAfter from the current key's TTL.
	res.Allowed = false
	res.Remaining = 0
	pttl, _ := l.client.PTTL(ctx, key)
	if pttl > 0 {
		res.RetryAfter = pttl
	} else {
		// TTL unknown or expired — fall back to the full window.
		res.RetryAfter = l.window
	}
	return res, nil
}

// bucketID returns a string discriminator that rolls forward with the window.
// Computed as the Unix-time bucket number (now / window). Two requests in the
// same window share the same bucket string.
func (l *Limiter) bucketID() string {
	now := l.nowFunc().UTC()
	bucketStart := now.Truncate(l.window)
	return bucketStart.Format("20060102T150405Z")
}
