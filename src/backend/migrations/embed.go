// Package migrations embeds the SQL migration files and applies them with goose.
// Embedding keeps migrations inside the binary, so deploys need no extra files.
package migrations

import (
	"context"
	"database/sql"
	"embed"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib" // database/sql driver "pgx"
	"github.com/pressly/goose/v3"
)

//go:embed *.sql
var files embed.FS

// Up applies all pending migrations.
func Up(ctx context.Context, dsn string) error {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("migrations: open: %w", err)
	}
	defer db.Close()

	goose.SetBaseFS(files)
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("migrations: dialect: %w", err)
	}
	if err := goose.UpContext(ctx, db, "."); err != nil {
		return fmt.Errorf("migrations: up: %w", err)
	}
	return nil
}
