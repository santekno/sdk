package redisx

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// ErrKeyNotFound is returned by GetJSON when the key does not exist.
var ErrKeyNotFound = errors.New("redisx: key not found")

// ErrLockNotAcquired is returned by Lock when another process holds the lock.
var ErrLockNotAcquired = errors.New("redisx: lock not acquired")

// RedisClient is the minimal interface that any Redis client must satisfy to
// work with redisx helpers. github.com/redis/go-redis/v9 satisfies this
// interface out of the box.
type RedisClient interface {
	// Get retrieves the string value for key. Returns ("", ErrKeyNotFound)
	// when the key does not exist.
	Get(ctx context.Context, key string) (string, error)
	// Set stores value for key with the given TTL (0 = no expiry).
	Set(ctx context.Context, key string, value any, ttl time.Duration) error
	// Del deletes one or more keys.
	Del(ctx context.Context, keys ...string) error
	// SetNX stores value for key only if the key does not exist.
	// Returns true if the key was set, false if it already existed.
	SetNX(ctx context.Context, key string, value any, ttl time.Duration) (bool, error)
}

// Client wraps a RedisClient with JSON marshalling and locking helpers.
type Client struct {
	rc RedisClient
}

// New returns a Client wrapping rc. rc must not be nil.
func New(rc RedisClient) *Client {
	if rc == nil {
		panic("redisx.New: rc must not be nil")
	}
	return &Client{rc: rc}
}

// GetJSON retrieves the JSON-encoded value stored at key and decodes it into dst.
// Returns ErrKeyNotFound if the key does not exist.
func (c *Client) GetJSON(ctx context.Context, key string, dst any) error {
	raw, err := c.rc.Get(ctx, key)
	if err != nil {
		return fmt.Errorf("redisx: get %q: %w", key, err)
	}
	if err := json.Unmarshal([]byte(raw), dst); err != nil {
		return fmt.Errorf("redisx: unmarshal %q: %w", key, err)
	}
	return nil
}

// SetJSON JSON-encodes value and stores it at key with the given TTL.
func (c *Client) SetJSON(ctx context.Context, key string, value any, ttl time.Duration) error {
	b, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("redisx: marshal for %q: %w", key, err)
	}
	if err := c.rc.Set(ctx, key, string(b), ttl); err != nil {
		return fmt.Errorf("redisx: set %q: %w", key, err)
	}
	return nil
}

// Lock acquires a distributed lock at key with the given TTL using SET NX.
// Returns an unlock function on success, or ErrLockNotAcquired if the lock
// is held by another process.
func (c *Client) Lock(ctx context.Context, key string, ttl time.Duration) (unlock func(), err error) {
	token := randomToken()
	lockKey := "lock:" + key

	ok, err := c.rc.SetNX(ctx, lockKey, token, ttl)
	if err != nil {
		return nil, fmt.Errorf("redisx: lock %q: %w", key, err)
	}
	if !ok {
		return nil, ErrLockNotAcquired
	}

	return func() {
		_ = c.rc.Del(context.Background(), lockKey)
	}, nil
}

func randomToken() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
