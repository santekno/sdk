package redisx_test

import (
	"context"
	"testing"
	"time"

	"github.com/santekno/sdk/redisx"
	"github.com/santekno/sdk/testx"
)

// mockRedis is a simple in-memory implementation of redisx.RedisClient for unit tests.
type mockRedis struct {
	data map[string]string
}

func newMock() *mockRedis { return &mockRedis{data: make(map[string]string)} }

func (m *mockRedis) Get(_ context.Context, key string) (string, error) {
	v, ok := m.data[key]
	if !ok {
		return "", redisx.ErrKeyNotFound
	}
	return v, nil
}

func (m *mockRedis) Set(_ context.Context, key string, value any, _ time.Duration) error {
	m.data[key] = value.(string)
	return nil
}

func (m *mockRedis) Del(_ context.Context, keys ...string) error {
	for _, k := range keys {
		delete(m.data, k)
	}
	return nil
}

func (m *mockRedis) SetNX(_ context.Context, key string, value any, _ time.Duration) (bool, error) {
	if _, ok := m.data[key]; ok {
		return false, nil
	}
	m.data[key] = value.(string)
	return true, nil
}

func TestGetSetJSON(t *testing.T) {
	c := redisx.New(newMock())
	type payload struct{ Name string }

	if err := c.SetJSON(context.Background(), "k", payload{"alice"}, 0); err != nil {
		t.Fatalf("SetJSON: %v", err)
	}

	var got payload
	if err := c.GetJSON(context.Background(), "k", &got); err != nil {
		t.Fatalf("GetJSON: %v", err)
	}
	if got.Name != "alice" {
		t.Errorf("Name = %q, want alice", got.Name)
	}
}

func TestGetJSON_NotFound(t *testing.T) {
	c := redisx.New(newMock())
	var v any
	err := c.GetJSON(context.Background(), "missing", &v)
	if err == nil {
		t.Error("expected error for missing key")
	}
}

func TestLock(t *testing.T) {
	c := redisx.New(newMock())

	unlock, err := c.Lock(context.Background(), "res", time.Second)
	if err != nil {
		t.Fatalf("Lock: %v", err)
	}

	// Second lock attempt should fail.
	_, err2 := c.Lock(context.Background(), "res", time.Second)
	if err2 != redisx.ErrLockNotAcquired {
		t.Errorf("second lock = %v, want ErrLockNotAcquired", err2)
	}

	// After unlock, should succeed again.
	unlock()
	_, err3 := c.Lock(context.Background(), "res", time.Second)
	if err3 != nil {
		t.Errorf("lock after unlock: %v", err3)
	}
}

func TestIntegration(t *testing.T) {
	_ = testx.RedisContainer(t) // skips if REDIS_ADDR not set
	t.Log("Integration test skipped: provide REDIS_ADDR for real Redis test")
}
