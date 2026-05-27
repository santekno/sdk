// redis-cache-pattern demonstrates the cache-aside pattern using redisx.Client.
//
// The example uses an in-memory mock so no real Redis server is needed.
// Replace the mock with a real go-redis client for production use:
//
//	import goredis "github.com/redis/go-redis/v9"
//	rc := goredis.NewClient(&goredis.Options{Addr: "localhost:6379"})
//	client := redisx.New(rc)
package main

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/santekno/sdk/redisx"
)

// ── in-memory mock (replaces go-redis in production) ─────────────────────────

type memRedis struct {
	mu   sync.RWMutex
	data map[string]string
}

func newMemRedis() *memRedis { return &memRedis{data: make(map[string]string)} }

func (m *memRedis) Get(_ context.Context, key string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	v, ok := m.data[key]
	if !ok {
		return "", redisx.ErrKeyNotFound
	}
	return v, nil
}

func (m *memRedis) Set(_ context.Context, key string, value any, _ time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[key] = fmt.Sprint(value)
	return nil
}

func (m *memRedis) Del(_ context.Context, keys ...string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, k := range keys {
		delete(m.data, k)
	}
	return nil
}

func (m *memRedis) SetNX(_ context.Context, key string, value any, _ time.Duration) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.data[key]; ok {
		return false, nil
	}
	m.data[key] = fmt.Sprint(value)
	return true, nil
}

// ── domain types ──────────────────────────────────────────────────────────────

type User struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// ── cache-aside helper ────────────────────────────────────────────────────────

func getUser(ctx context.Context, rc *redisx.Client, id int) (User, error) {
	key := fmt.Sprintf("user:%d", id)

	var u User
	err := rc.GetJSON(ctx, key, &u)
	if err == nil {
		fmt.Printf("[cache HIT]  user %d → %s\n", id, u.Name)
		return u, nil
	}
	if !errors.Is(err, redisx.ErrKeyNotFound) {
		return User{}, fmt.Errorf("cache get: %w", err)
	}

	// Cache miss — fetch from "database".
	u = User{ID: id, Name: fmt.Sprintf("User #%d", id)}
	fmt.Printf("[cache MISS] user %d → fetched from DB\n", id)

	if err := rc.SetJSON(ctx, key, u, 30*time.Second); err != nil {
		return User{}, fmt.Errorf("cache set: %w", err)
	}
	return u, nil
}

func main() {
	ctx := context.Background()
	client := redisx.New(newMemRedis())

	// First call — cache miss, populates cache.
	u, _ := getUser(ctx, client, 42)
	fmt.Printf("Got: %+v\n\n", u)

	// Second call — cache hit.
	u, _ = getUser(ctx, client, 42)
	fmt.Printf("Got: %+v\n\n", u)

	// Distributed lock demo.
	unlock, err := client.Lock(ctx, "resource:42", 5*time.Second)
	if err != nil {
		fmt.Printf("Lock failed: %v\n", err)
		return
	}
	fmt.Println("Lock acquired")

	_, err = client.Lock(ctx, "resource:42", 5*time.Second)
	if errors.Is(err, redisx.ErrLockNotAcquired) {
		fmt.Println("Second lock correctly rejected")
	}

	unlock()
	fmt.Println("Lock released")
}
