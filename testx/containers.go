package testx

import (
	"net"
	"os"
	"testing"
	"time"
)

// PostgresContainer returns a PostgreSQL DSN for integration tests.
//
// Checks the POSTGRES_DSN environment variable. When full testcontainers-go
// support is added (Phase 8), this will auto-spin a container if the variable
// is not set. For now, tests that need Postgres must set POSTGRES_DSN.
func PostgresContainer(t testing.TB) string {
	t.Helper()
	dsn := postgresEnvDSN()
	if dsn == "" {
		t.Skip("PostgresContainer: set POSTGRES_DSN env var to run this test")
	}
	return dsn
}

// RedisContainer returns a Redis address (host:port) for integration tests.
//
// Checks the REDIS_ADDR environment variable. When full testcontainers-go
// support is added (Phase 8), this will auto-spin a container if the variable
// is not set.
func RedisContainer(t testing.TB) string {
	t.Helper()
	addr := redisEnvAddr()
	if addr == "" {
		t.Skip("RedisContainer: set REDIS_ADDR env var to run this test")
	}
	if err := waitForTCP(addr, 5*time.Second); err != nil {
		t.Skipf("RedisContainer: %v", err)
	}
	return addr
}

func postgresEnvDSN() string  { return os.Getenv("POSTGRES_DSN") }
func redisEnvAddr() string    { return os.Getenv("REDIS_ADDR") }

func waitForTCP(addr string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, time.Second)
		if err == nil {
			_ = conn.Close()
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}
	return nil // skip rather than hard-fail if service not running
}
