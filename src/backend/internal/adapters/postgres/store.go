// Package postgres implements the ports.Store and repository interfaces using
// pgx. Repositories receive a small querier interface so the exact same code
// runs against either the connection pool or a transaction.
package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/youmei295/something-something/src/backend/internal/domain"
	"github.com/youmei295/something-something/src/backend/internal/ports"
)

// querier is satisfied by both *pgxpool.Pool and pgx.Tx.
type querier interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// Store owns the connection pool and hands out repositories.
type Store struct {
	pool *pgxpool.Pool
}

// New opens a pool and verifies connectivity.
func New(ctx context.Context, dsn string) (*Store, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("postgres: parse dsn: %w", err)
	}
	cfg.MaxConns = 10
	cfg.MinConns = 1
	cfg.MaxConnLifetime = time.Hour

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("postgres: connect: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("postgres: ping: %w", err)
	}
	return &Store{pool: pool}, nil
}

// Pool exposes the underlying pool for the migration command.
func (s *Store) Pool() *pgxpool.Pool { return s.pool }

func (s *Store) Repos() ports.Repos { return s.repos(s.pool) }

func (s *Store) repos(q querier) ports.Repos {
	return ports.Repos{
		Users:         &userRepo{q: q},
		Sessions:      &sessionRepo{q: q},
		Conversations: &conversationRepo{q: q},
		Messages:      &messageRepo{q: q},
		Attachments:   &attachmentRepo{q: q},
		Jobs:          &jobRepo{q: q},
	}
}

// WithinTx runs fn inside a transaction, committing on success and rolling back
// on any error. Nested repository calls share the transaction.
func (s *Store) WithinTx(ctx context.Context, fn func(ports.Repos) error) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("postgres: begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := fn(s.repos(tx)); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("postgres: commit: %w", err)
	}
	return nil
}

func (s *Store) Ping(ctx context.Context) error { return s.pool.Ping(ctx) }
func (s *Store) Close()                         { s.pool.Close() }

// mapErr translates low-level pgx errors into domain sentinel errors so the
// rest of the application never depends on the database driver.
func mapErr(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrNotFound
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return domain.ErrConflict
	}
	return err
}

var _ ports.Store = (*Store)(nil)
