// postgres-init demonstrates opening a PostgreSQL connection using sqlx.Open
// with sensible connection pool defaults and a context-aware health check.
//
// Run:
//
//	POSTGRES_DSN="postgres://user:pass@localhost/mydb?sslmode=disable" go run .
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/santekno/sdk/sqlx"
)

func main() {
	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		log.Fatal("POSTGRES_DSN environment variable is required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db, err := sqlx.Open(ctx, sqlx.Config{
		Driver:          "postgres",
		DSN:             dsn,
		MaxOpenConns:    10,
		MaxIdleConns:    3,
		ConnMaxLifetime: 5 * time.Minute,
	})
	if err != nil {
		log.Fatalf("open: %v", err)
	}
	defer db.Close()

	if err := db.PingContext(ctx); err != nil {
		log.Fatalf("ping: %v", err)
	}

	var version string
	row := db.QueryRowContext(ctx, "SELECT version()")
	if err := row.Scan(&version); err != nil {
		log.Fatalf("query: %v", err)
	}

	fmt.Printf("Connected!\n%s\n", version)
}
