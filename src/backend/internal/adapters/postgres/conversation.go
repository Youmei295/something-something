package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/youmei295/something-something/src/backend/internal/domain"
	"github.com/youmei295/something-something/src/backend/internal/ports"
)

type conversationRepo struct{ q querier }

const conversationColumns = `id, subject, status, visitor_name, visitor_email, visitor_token, unread, created_at, last_activity_at`

func (r *conversationRepo) Create(ctx context.Context, c *domain.Conversation) error {
	const q = `
		INSERT INTO conversations
			(id, subject, status, visitor_name, visitor_email, visitor_token, unread, created_at, last_activity_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`
	_, err := r.q.Exec(ctx, q, c.ID, c.Subject, c.Status, c.VisitorName, c.VisitorEmail,
		c.VisitorToken, c.Unread, c.CreatedAt, c.LastActivityAt)
	return mapErr(err)
}

func (r *conversationRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Conversation, error) {
	q := `SELECT ` + conversationColumns + ` FROM conversations WHERE id = $1`
	return scanConversation(r.q.QueryRow(ctx, q, id))
}

func (r *conversationRepo) GetByVisitorToken(ctx context.Context, token string) (*domain.Conversation, error) {
	q := `SELECT ` + conversationColumns + ` FROM conversations WHERE visitor_token = $1`
	return scanConversation(r.q.QueryRow(ctx, q, token))
}

func (r *conversationRepo) List(ctx context.Context, f ports.ListConversationsFilter) ([]domain.Conversation, int, error) {
	if f.Limit <= 0 || f.Limit > 100 {
		f.Limit = 25
	}

	var args []any
	var where []string

	if f.Status != "" && f.Status != ports.FilterAll {
		args = append(args, string(f.Status))
		where = append(where, fmt.Sprintf("status = $%d", len(args)))
	}
	if s := strings.TrimSpace(f.Search); s != "" {
		args = append(args, "%"+s+"%")
		p := fmt.Sprintf("$%d", len(args))
		where = append(where, fmt.Sprintf("(subject ILIKE %s OR visitor_name ILIKE %s OR visitor_email ILIKE %s)", p, p, p))
	}

	clause := ""
	if len(where) > 0 {
		clause = " WHERE " + strings.Join(where, " AND ")
	}

	var total int
	if err := r.q.QueryRow(ctx, `SELECT count(*) FROM conversations`+clause, args...).Scan(&total); err != nil {
		return nil, 0, mapErr(err)
	}

	args = append(args, f.Limit, f.Offset)
	query := fmt.Sprintf(
		`SELECT %s FROM conversations%s ORDER BY last_activity_at DESC LIMIT $%d OFFSET $%d`,
		conversationColumns, clause, len(args)-1, len(args),
	)

	rows, err := r.q.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, mapErr(err)
	}
	defer rows.Close()

	out := make([]domain.Conversation, 0)
	for rows.Next() {
		c, err := scanConversation(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, *c)
	}
	return out, total, mapErr(rows.Err())
}

func (r *conversationRepo) Update(ctx context.Context, c *domain.Conversation) error {
	const q = `
		UPDATE conversations
		SET subject = $2, status = $3, unread = $4, last_activity_at = $5
		WHERE id = $1`
	tag, err := r.q.Exec(ctx, q, c.ID, c.Subject, c.Status, c.Unread, c.LastActivityAt)
	if err != nil {
		return mapErr(err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *conversationRepo) Delete(ctx context.Context, id uuid.UUID) error {
	tag, err := r.q.Exec(ctx, `DELETE FROM conversations WHERE id = $1`, id)
	if err != nil {
		return mapErr(err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func scanConversation(row interface {
	Scan(dest ...any) error
}) (*domain.Conversation, error) {
	var c domain.Conversation
	err := row.Scan(&c.ID, &c.Subject, &c.Status, &c.VisitorName, &c.VisitorEmail,
		&c.VisitorToken, &c.Unread, &c.CreatedAt, &c.LastActivityAt)
	if err != nil {
		return nil, mapErr(err)
	}
	return &c, nil
}
