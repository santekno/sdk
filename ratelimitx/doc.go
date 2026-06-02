// Package ratelimitx provides a fixed-window per-key rate limiter backed by
// Redis. It is dependency-free: bring your own Redis client (e.g.
// github.com/redis/go-redis/v9) and satisfy the [Client] interface.
//
// # Design
//
// ratelimitx implements a simple **fixed-window counter** keyed on the caller-
// chosen identifier (typically an IP address or user ID). Each Allow call
// increments a counter under a Redis key shaped like `<prefix>:<id>:<bucket>`,
// where `<bucket>` is a time-derived discriminator that rolls forward with
// the window. Counters that exceed the configured limit cause Allow to return
// `allowed=false` along with a Retry-After hint computed from the key's TTL.
//
// Fixed-window (vs sliding-window or token-bucket): chosen for operational
// simplicity at the scale ratelimitx targets — small services with modest
// per-key traffic. For high-throughput or precision-critical workloads, use
// a true token bucket. Fixed-window has a known edge case (a burst can fire
// 2× the limit across a window boundary); this is acceptable for the
// "fairness, not security" use case.
//
// # Usage
//
//	import (
//	    "github.com/redis/go-redis/v9"
//	    "github.com/santekno/sdk/ratelimitx"
//	)
//
//	rc := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
//	limiter := ratelimitx.New(rc,
//	    ratelimitx.WithLimit(10),
//	    ratelimitx.WithWindow(24*time.Hour),
//	    ratelimitx.WithPrefix("ai:rl"),
//	)
//
//	res, err := limiter.Allow(ctx, clientIP)
//	if err != nil { ... }
//	if !res.Allowed {
//	    // Return 429 with Retry-After: res.RetryAfter
//	    return
//	}
//
// # Adapter
//
// The official go-redis client doesn't directly match the [Client] interface
// signatures (return types differ). A thin adapter is documented in the
// example_test.go file and trivially trivial to write.
package ratelimitx
