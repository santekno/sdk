// Package sqlx wraps database/sql with structured logging, OTel tracing,
// and connection-pool defaults tuned for production workloads.
//
// It is a thin, dependency-free layer — no ORM, no query builder. Bring
// your own driver (e.g. lib/pq, go-sqlite3) via a blank import in your
// application binary.
//
// # Quick Start
//
//	cfg := sqlx.Config{
//	    Driver: "postgres",
//	    DSN:    os.Getenv("DATABASE_URL"),
//	}
//	db, err := sqlx.Open(ctx, cfg)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer db.Close()
//
//	// Use the embedded *sql.DB directly.
//	rows, err := db.QueryContext(ctx, "SELECT id, name FROM users")
package sqlx
