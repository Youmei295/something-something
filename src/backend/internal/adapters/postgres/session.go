package postgres

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/youmei295/something-something/src/backend/internal/domain"
)

type sessionRepo struct{ q querier }

func (r *sessionRepo) Create(ctx context.Context, s *domain.Session) error {
	const q = `
		INSERT INTO sessions (id, user_id, expires_at, created_at)
		VALUES ($1, $2, $3, $4)`
	if s.CreatedAt.IsZero() {
		s.CreatedAt = time.Now().UTC()
	}
	_, err := r.q.Exec(ctx, q, s.ID, s.UserID, s.ExpiresAt, s.CreatedAt)
	return mapErr(err)
}

func (r *sessionRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Session, error) {
	const q = `SELECT id, user_id, expires_at, created_at FROM sessions WHERE id = $1`
	var s domain.Session
	err := r.q.QueryRow(ctx, q, id).Scan(&s.ID, &s.UserID, &s.ExpiresAt, &s.CreatedAt)
	if err != nil {
		return nil, mapErr(err)
	}
	return &s, nil
}

func (r *sessionRepo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.q.Exec(ctx, `DELETE FROM sessions WHERE id = $1`, id)
	return mapErr(err)
}

func (r *sessionRepo) DeleteExpired(ctx context.Context, before time.Time) (int64, error) {
	tag, err := r.q.Exec(ctx, `DELETE FROM sessions WHERE expires_at < $1`, before)
	if err != nil {
		return 0, mapErr(err)
	}
	return tag.RowsAffected(), nil
}
