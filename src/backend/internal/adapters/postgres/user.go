package postgres

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/youmei295/something-something/src/backend/internal/domain"
)

type userRepo struct{ q querier }

func (r *userRepo) Create(ctx context.Context, u *domain.User) error {
	const q = `
		INSERT INTO users (id, email, password_hash, role, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)`
	now := time.Now().UTC()
	if u.CreatedAt.IsZero() {
		u.CreatedAt = now
	}
	if u.UpdatedAt.IsZero() {
		u.UpdatedAt = now
	}
	_, err := r.q.Exec(ctx, q, u.ID, u.Email, u.PasswordHash, u.Role, u.CreatedAt, u.UpdatedAt)
	return mapErr(err)
}

func (r *userRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	const q = `
		SELECT id, email, password_hash, role, created_at, updated_at
		FROM users WHERE email = $1`
	return scanUser(r.q.QueryRow(ctx, q, email))
}

func (r *userRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	const q = `
		SELECT id, email, password_hash, role, created_at, updated_at
		FROM users WHERE id = $1`
	return scanUser(r.q.QueryRow(ctx, q, id))
}

func (r *userRepo) Count(ctx context.Context) (int, error) {
	var n int
	err := r.q.QueryRow(ctx, `SELECT count(*) FROM users`).Scan(&n)
	return n, mapErr(err)
}

func scanUser(row interface {
	Scan(dest ...any) error
}) (*domain.User, error) {
	var u domain.User
	err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Role, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, mapErr(err)
	}
	return &u, nil
}
