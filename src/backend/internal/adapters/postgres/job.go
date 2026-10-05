package postgres

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/youmei295/something-something/src/backend/internal/domain"
)

type jobRepo struct{ q querier }

func (r *jobRepo) Enqueue(ctx context.Context, j *domain.Job) error {
	const q = `
		INSERT INTO jobs (id, type, payload, status, attempts, max_attempts, run_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`
	now := time.Now().UTC()
	if j.CreatedAt.IsZero() {
		j.CreatedAt = now
	}
	if j.UpdatedAt.IsZero() {
		j.UpdatedAt = now
	}
	if j.MaxAttempts == 0 {
		j.MaxAttempts = 5
	}
	_, err := r.q.Exec(ctx, q, j.ID, j.Type, j.Payload, j.Status, j.Attempts, j.MaxAttempts, j.RunAt, j.CreatedAt, j.UpdatedAt)
	return mapErr(err)
}

func (r *jobRepo) Dequeue(ctx context.Context, types []string, limit int) ([]domain.Job, error) {
	// A nil slice encodes as SQL NULL, which would make the cardinality check
	// below evaluate to NULL and claim nothing. Use an empty array instead.
	if types == nil {
		types = []string{}
	}
	const q = `
		WITH claimed AS (
			SELECT id FROM jobs
			WHERE status = 'pending'
			  AND run_at <= now()
			  AND (cardinality($1::text[]) = 0 OR type = ANY($1))
			ORDER BY run_at ASC
			LIMIT $2
			FOR UPDATE SKIP LOCKED
		)
		UPDATE jobs
		SET status = 'running', attempts = attempts + 1, updated_at = now()
		WHERE id IN (SELECT id FROM claimed)
		RETURNING id, type, payload, status, attempts, max_attempts, run_at, last_error, created_at, updated_at`

	rows, err := r.q.Query(ctx, q, types, limit)
	if err != nil {
		return nil, mapErr(err)
	}
	defer rows.Close()

	out := make([]domain.Job, 0)
	for rows.Next() {
		var j domain.Job
		if err := rows.Scan(&j.ID, &j.Type, &j.Payload, &j.Status, &j.Attempts,
			&j.MaxAttempts, &j.RunAt, &j.LastError, &j.CreatedAt, &j.UpdatedAt); err != nil {
			return nil, mapErr(err)
		}
		out = append(out, j)
	}
	return out, mapErr(rows.Err())
}

func (r *jobRepo) Complete(ctx context.Context, id uuid.UUID) error {
	_, err := r.q.Exec(ctx, `UPDATE jobs SET status = 'done', updated_at = now() WHERE id = $1`, id)
	return mapErr(err)
}

func (r *jobRepo) Fail(ctx context.Context, id uuid.UUID, errMsg string, retryAt time.Time) error {
	const q = `
		UPDATE jobs
		SET status = CASE WHEN attempts >= max_attempts THEN 'failed' ELSE 'pending' END,
		    last_error = $2,
		    run_at = $3,
		    updated_at = now()
		WHERE id = $1`
	_, err := r.q.Exec(ctx, q, id, errMsg, retryAt)
	return mapErr(err)
}
