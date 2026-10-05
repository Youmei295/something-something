package app

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/youmei295/something-something/src/backend/internal/domain"
	"github.com/youmei295/something-something/src/backend/internal/ports"
)

// SubmissionService implements the visitor-facing "send a message" use case.
type SubmissionService struct {
	store          ports.Store
	ids            ports.IDGenerator
	tokens         ports.TokenGenerator
	clock          ports.Clock
	maxAttachments int
}

func NewSubmissionService(store ports.Store, ids ports.IDGenerator, tokens ports.TokenGenerator, clock ports.Clock, maxAttachments int) *SubmissionService {
	return &SubmissionService{store: store, ids: ids, tokens: tokens, clock: clock, maxAttachments: maxAttachments}
}

// SubmitInput is the raw data of a visitor submission.
type SubmitInput struct {
	Name          string
	Email         string
	Subject       string
	Body          string
	AttachmentIDs []uuid.UUID
}

// SubmitResult references the created conversation and its visitor token.
type SubmitResult struct {
	ConversationID uuid.UUID
	VisitorToken   string
	CreatedAt      time.Time
}

// Submit validates input, then atomically creates a conversation, the inbound
// message, links any attachments, and enqueues follow-up jobs.
func (s *SubmissionService) Submit(ctx context.Context, in SubmitInput) (*SubmitResult, error) {
	in.Name = strings.TrimSpace(in.Name)
	in.Email = strings.ToLower(strings.TrimSpace(in.Email))
	in.Subject = strings.TrimSpace(in.Subject)
	in.Body = strings.TrimSpace(in.Body)

	v := NewValidate()
	v.Required("name", in.Name)
	v.MaxLen("name", in.Name, 120)
	v.Required("email", in.Email)
	v.Email("email", in.Email)
	v.MaxLen("email", in.Email, 254)
	v.Required("subject", in.Subject)
	v.MaxLen("subject", in.Subject, 200)
	v.Required("body", in.Body)
	v.MaxLen("body", in.Body, 20000)
	v.MaxItems("attachments", len(in.AttachmentIDs), s.maxAttachments)
	if err := v.Err(); err != nil {
		return nil, err
	}

	token, err := s.tokens.NewToken(32)
	if err != nil {
		return nil, err
	}

	now := s.clock.Now()
	conversation := &domain.Conversation{
		ID:             s.ids.New(),
		Subject:        in.Subject,
		Status:         domain.StatusOpen,
		VisitorName:    in.Name,
		VisitorEmail:   in.Email,
		VisitorToken:   token,
		Unread:         true,
		CreatedAt:      now,
		LastActivityAt: now,
	}
	message := &domain.Message{
		ID:             s.ids.New(),
		ConversationID: conversation.ID,
		Direction:      domain.DirectionInbound,
		Body:           in.Body,
		FromAddr:       in.Email,
		CreatedAt:      now,
	}

	err = s.store.WithinTx(ctx, func(repos ports.Repos) error {
		if err := repos.Conversations.Create(ctx, conversation); err != nil {
			return err
		}
		if err := repos.Messages.Create(ctx, message); err != nil {
			return err
		}
		if len(in.AttachmentIDs) > 0 {
			if err := repos.Attachments.LinkToMessage(ctx, in.AttachmentIDs, conversation.ID, message.ID); err != nil {
				return err
			}
		}
		if err := enqueue(ctx, s.ids, repos, domain.JobNotifyHost, NotifyHostPayload{
			ConversationID: conversation.ID, MessageID: message.ID,
		}); err != nil {
			return err
		}
		for _, id := range in.AttachmentIDs {
			if err := enqueue(ctx, s.ids, repos, domain.JobScanAttachment, ScanAttachmentPayload{AttachmentID: id}); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &SubmitResult{ConversationID: conversation.ID, VisitorToken: token, CreatedAt: now}, nil
}
