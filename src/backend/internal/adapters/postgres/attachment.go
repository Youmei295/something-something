package postgres

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/youmei295/something-something/src/backend/internal/domain"
)

type attachmentRepo struct{ q querier }

const attachmentColumns = `id, conversation_id, message_id, filename, content_type, size, storage_key, sha256, upload_status, scan_status, created_at`

func (r *attachmentRepo) Create(ctx context.Context, a *domain.Attachment) error {
	const q = `
		INSERT INTO attachments
			(id, conversation_id, message_id, filename, content_type, size, storage_key, sha256, upload_status, scan_status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`
	_, err := r.q.Exec(ctx, q, a.ID, a.ConversationID, a.MessageID, a.Filename, a.ContentType,
		a.Size, a.StorageKey, a.SHA256, a.UploadStatus, a.ScanStatus, a.CreatedAt)
	return mapErr(err)
}

func (r *attachmentRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Attachment, error) {
	q := `SELECT ` + attachmentColumns + ` FROM attachments WHERE id = $1`
	return scanAttachment(r.q.QueryRow(ctx, q, id))
}

func (r *attachmentRepo) ListByConversation(ctx context.Context, conversationID uuid.UUID) ([]domain.Attachment, error) {
	q := `SELECT ` + attachmentColumns + ` FROM attachments WHERE conversation_id = $1 ORDER BY created_at ASC`
	rows, err := r.q.Query(ctx, q, conversationID)
	if err != nil {
		return nil, mapErr(err)
	}
	defer rows.Close()

	out := make([]domain.Attachment, 0)
	for rows.Next() {
		a, err := scanAttachment(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *a)
	}
	return out, mapErr(rows.Err())
}

func (r *attachmentRepo) LinkToMessage(ctx context.Context, ids []uuid.UUID, conversationID, messageID uuid.UUID) error {
	if len(ids) == 0 {
		return nil
	}
	const q = `
		UPDATE attachments
		SET conversation_id = $1, message_id = $2, upload_status = 'linked'
		WHERE id = ANY($3) AND upload_status IN ('pending','complete')`
	_, err := r.q.Exec(ctx, q, conversationID, messageID, ids)
	return mapErr(err)
}

func (r *attachmentRepo) MarkUploaded(ctx context.Context, id uuid.UUID, size int64) error {
	const q = `
		UPDATE attachments
		SET upload_status = 'complete', size = $2
		WHERE id = $1 AND upload_status = 'pending'`
	tag, err := r.q.Exec(ctx, q, id, size)
	if err != nil {
		return mapErr(err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *attachmentRepo) MarkScanStatus(ctx context.Context, id uuid.UUID, status domain.ScanStatus) error {
	_, err := r.q.Exec(ctx, `UPDATE attachments SET scan_status = $2 WHERE id = $1`, id, status)
	return mapErr(err)
}

func (r *attachmentRepo) ListOrphans(ctx context.Context, before time.Time) ([]domain.Attachment, error) {
	const q = `SELECT ` + attachmentColumns + `
		FROM attachments
		WHERE upload_status IN ('pending','complete') AND created_at < $1
		ORDER BY created_at ASC
		LIMIT 100`
	rows, err := r.q.Query(ctx, q, before)
	if err != nil {
		return nil, mapErr(err)
	}
	defer rows.Close()

	out := make([]domain.Attachment, 0)
	for rows.Next() {
		a, err := scanAttachment(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *a)
	}
	return out, mapErr(rows.Err())
}

func (r *attachmentRepo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.q.Exec(ctx, `DELETE FROM attachments WHERE id = $1`, id)
	return mapErr(err)
}

func scanAttachment(row interface {
	Scan(dest ...any) error
}) (*domain.Attachment, error) {
	var a domain.Attachment
	err := row.Scan(&a.ID, &a.ConversationID, &a.MessageID, &a.Filename, &a.ContentType,
		&a.Size, &a.StorageKey, &a.SHA256, &a.UploadStatus, &a.ScanStatus, &a.CreatedAt)
	if err != nil {
		return nil, mapErr(err)
	}
	return &a, nil
}
