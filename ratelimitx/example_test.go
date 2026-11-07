package ratelimitx_test

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/santekno/sdk/ratelimitx"
)

// memClient is an in-memory ratelimitx.Client. In production, wrap your Redis
// client instead: go-redis returns *redis.IntCmd and friends, so an adapter
// maps rc.Incr(ctx, key).Result(), rc.Expire(ctx, key, ttl).Err() and
// rc.PTTL(ctx, key).Result() onto the three Client methods.
type memClient struct {
	mu sync.Mutex
	n  map[string]int64
}

func (m *memClient) Incr(_ context.Context, key string) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.n[key]++
	return m.n[key], nil
}

func (m *memClient) Expire(context.Context, string, time.Duration) error { return nil }

func (m *memClient) PTTL(context.Context, string) (time.Duration, error) {
	return time.Minute, nil
}

func Example() {
	limiter := ratelimitx.New(&memClient{n: map[string]int64{}},
		ratelimitx.WithLimit(2),
		ratelimitx.WithWindow(time.Hour),
		ratelimitx.WithPrefix("demo"),
	)
	for range 3 {
		res, _ := limiter.Allow(context.Background(), "client-key")
		fmt.Println(res.Allowed)
	}
	// Output:
	// true
	// true
	// false
}
