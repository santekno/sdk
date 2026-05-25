//go:build ignore

// This file is intentionally excluded from normal builds.
//
// OpenGORM wraps a sqlx.Config to open a gorm.DB using the existing
// *sql.DB connection pool. To use it, import gorm.io/gorm in your
// application and adapt the snippet below:
//
//	import (
//	    "gorm.io/driver/postgres"
//	    "gorm.io/gorm"
//	    "github.com/santekno/sdk/sqlx"
//	)
//
//	db, _ := sqlx.Open(ctx, cfg)
//	gdb, err := gorm.Open(postgres.New(postgres.Config{Conn: db.DB}), &gorm.Config{})
//
// Adding gorm as a direct dependency of github.com/santekno/sdk would force
// all SDK users to download it even if they never use GORM. Use the snippet
// above in your application module instead.
package sqlx
