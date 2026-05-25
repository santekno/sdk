package sqlx

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/santekno/sdk/logx"
	"github.com/santekno/sdk/tracex"
)

// Config holds connection parameters for Open.
type Config struct {
	// Driver is the database driver name registered via sql.Register
	// (e.g., "postgres", "mysql", "sqlite3").
	Driver string

	// DSN is the data source name / connection string.
	DSN string

	// MaxOpenConns sets the maximum number of open connections.
	// Defaults to 25 if zero.
	MaxOpenConns int

	// MaxIdleConns sets the maximum number of idle connections.
	// Defaults to 5 if zero.
	MaxIdleConns int

	// ConnMaxLifetime sets the maximum amount of time a connection may be reused.
	// Defaults to 5 minutes if zero.
	ConnMaxLifetime time.Duration

	// Logger is an optional structured logger for query tracing.
	Logger logx.Logger

	// Tracer is an optional OTel-compatible tracer for span creation.
	Tracer tracex.Tracer
}

// DB wraps *sql.DB with logging and tracing capabilities.
type DB struct {
	*sql.DB
	cfg Config
}

// Open opens a database connection pool with the supplied Config and verifies
// connectivity via Ping. Returns an error if the driver is unknown or the
// first ping fails.
func Open(ctx context.Context, cfg Config) (*DB, error) {
	if cfg.Driver == "" {
		return nil, fmt.Errorf("sqlx: driver must not be empty")
	}
	if cfg.DSN == "" {
		return nil, fmt.Errorf("sqlx: DSN must not be empty")
	}

	db, err := sql.Open(cfg.Driver, cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("sqlx: open %s: %w", cfg.Driver, err)
	}

	// Apply pool defaults.
	maxOpen := cfg.MaxOpenConns
	if maxOpen == 0 {
		maxOpen = 25
	}
	maxIdle := cfg.MaxIdleConns
	if maxIdle == 0 {
		maxIdle = 5
	}
	lifetime := cfg.ConnMaxLifetime
	if lifetime == 0 {
		lifetime = 5 * time.Minute
	}

	db.SetMaxOpenConns(maxOpen)
	db.SetMaxIdleConns(maxIdle)
	db.SetConnMaxLifetime(lifetime)

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("sqlx: ping %s: %w", cfg.Driver, err)
	}

	if cfg.Logger != nil {
		cfg.Logger.Info("sqlx: database connected",
			"driver", cfg.Driver,
			"max_open", maxOpen,
			"max_idle", maxIdle,
		)
	}

	return &DB{DB: db, cfg: cfg}, nil
}

// MustOpen calls Open and panics if it returns an error. Intended for
// use in application main functions where a missing database is fatal.
func MustOpen(ctx context.Context, cfg Config) *DB {
	db, err := Open(ctx, cfg)
	if err != nil {
		panic(fmt.Sprintf("sqlx.MustOpen: %v", err))
	}
	return db
}

// Logger returns the configured logger, or a no-op logger if none was set.
func (db *DB) Logger() logx.Logger {
	if db.cfg.Logger != nil {
		return db.cfg.Logger
	}
	return logx.Noop()
}

// Tracer returns the configured tracer, or a no-op tracer if none was set.
func (db *DB) Tracer() tracex.Tracer {
	if db.cfg.Tracer != nil {
		return db.cfg.Tracer
	}
	return tracex.NoopTracer()
}
