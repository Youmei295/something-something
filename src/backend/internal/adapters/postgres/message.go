package postgres

import (
	"context"

	"github.com/google/uuid"

	"github.com/youmei295/something-something/src/backend/internal/domain"
)

type messageRepo struct{ q querier }

const messageColumns = `id, conversation_id, direction, body, from_addr, to_addr, email_message_id, in_reply_to, created_at`

func (r *messageRepo) Create(ctx context.Context, m *domain.Message) error {
	const q = `
		INSERT INTO messages
			(id, conversation_id, direction, body, from_addr, to_addr, email_message_id, in_reply_to, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`
	_, err := r.q.Exec(ctx, q, m.ID, m.ConversationID, m.Direction, m.Body, m.FromAddr,
		m.ToAddr, m.EmailMessageID, m.InReplyTo, m.CreatedAt)
	return mapErr(err)
}

func (r *messageRepo) ListByConversation(ctx context.Context, conversationID uuid.UUID) ([]domain.Message, error) {
	q := `SELECT ` + messageColumns + ` FROM messages WHERE conversation_id = $1 ORDER BY created_at ASC`
	rows, err := r.q.Query(ctx, q, conversationID)
	if err != nil {
		return nil, mapErr(err)
	}
	defer rows.Close()

	out := make([]domain.Message, 0)
	for rows.Next() {
		var m domain.Message
		if err := rows.Scan(&m.ID, &m.ConversationID, &m.Direction, &m.Body, &m.FromAddr,
			&m.ToAddr, &m.EmailMessageID, &m.InReplyTo, &m.CreatedAt); err != nil {
			return nil, mapErr(err)
		}
		out = append(out, m)
	}
	return out, mapErr(rows.Err())
}

func (r *messageRepo) GetByEmailMessageID(ctx context.Context, emailMessageID string) (*domain.Message, error) {
	q := `SELECT ` + messageColumns + ` FROM messages WHERE email_message_id = $1`
	var m domain.Message
	err := r.q.QueryRow(ctx, q, emailMessageID).Scan(&m.ID, &m.ConversationID, &m.Direction,
		&m.Body, &m.FromAddr, &m.ToAddr, &m.EmailMessageID, &m.InReplyTo, &m.CreatedAt)
	if err != nil {
		return nil, mapErr(err)
	}
	return &m, nil
}
