package sqlx_test

import (
	"context"
	"testing"

	"github.com/santekno/sdk/sqlx"
	"github.com/santekno/sdk/testx"
)

func TestOpen_InvalidDriver(t *testing.T) {
	_, err := sqlx.Open(context.Background(), sqlx.Config{
		Driver: "nonexistent-driver",
		DSN:    "dsn",
	})
	if err == nil {
		t.Error("expected error for unknown driver")
	}
}

func TestOpen_EmptyDriver(t *testing.T) {
	_, err := sqlx.Open(context.Background(), sqlx.Config{DSN: "dsn"})
	if err == nil {
		t.Error("expected error for empty driver")
	}
}

func TestOpen_EmptyDSN(t *testing.T) {
	_, err := sqlx.Open(context.Background(), sqlx.Config{Driver: "postgres"})
	if err == nil {
		t.Error("expected error for empty DSN")
	}
}

func TestOpen_Integration(t *testing.T) {
	dsn := testx.PostgresContainer(t) // skips if POSTGRES_DSN not set

	db, err := sqlx.Open(context.Background(), sqlx.Config{
		Driver: "postgres",
		DSN:    dsn,
	})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer db.Close()

	if db.DB == nil {
		t.Error("expected non-nil *sql.DB")
	}
}
