package app

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/youmei295/something-something/src/backend/internal/domain"
	"github.com/youmei295/something-something/src/backend/internal/ports"
)

// Job payloads are stored as JSON so producers and consumers evolve together
// while remaining decoupled through the job table (and, later, the event bus).

type NotifyHostPayload struct {
	ConversationID uuid.UUID `json:"conversation_id"`
	MessageID      uuid.UUID `json:"message_id"`
}

type SendVisitorMailPayload struct {
	ConversationID uuid.UUID `json:"conversation_id"`
	MessageID      uuid.UUID `json:"message_id"`
}

type ScanAttachmentPayload struct {
	AttachmentID uuid.UUID `json:"attachment_id"`
}

type CleanupUploadsPayload struct {
	OlderThan time.Duration `json:"older_than"`
}

func newJob(ids ports.IDGenerator, jobType string, payload any) (*domain.Job, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return &domain.Job{
		ID:          ids.New(),
		Type:        jobType,
		Payload:     raw,
		Status:      domain.JobPending,
		MaxAttempts: 5,
		RunAt:       time.Now().UTC(),
	}, nil
}

func enqueue(ctx context.Context, ids ports.IDGenerator, repos ports.Repos, jobType string, payload any) error {
	job, err := newJob(ids, jobType, payload)
	if err != nil {
		return err
	}
	return repos.Jobs.Enqueue(ctx, job)
}
