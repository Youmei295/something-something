package app

import (
	"context"
	"strings"

	"github.com/google/uuid"

	"github.com/youmei295/something-something/src/backend/internal/domain"
	"github.com/youmei295/something-something/src/backend/internal/ports"
)

// ConversationService implements the host-facing inbox use cases.
type ConversationService struct {
	store          ports.Store
	ids            ports.IDGenerator
	clock          ports.Clock
	hostEmail      string
	maxAttachments int
}

func NewConversationService(store ports.Store, ids ports.IDGenerator, clock ports.Clock, hostEmail string, maxAttachments int) *ConversationService {
	return &ConversationService{store: store, ids: ids, clock: clock, hostEmail: hostEmail, maxAttachments: maxAttachments}
}

// ConversationDetail is a conversation together with its messages and files.
type ConversationDetail struct {
	Conversation domain.Conversation
	Messages     []domain.Message
	Attachments  []domain.Attachment
}

// List returns a page of conversations and the total count.
func (s *ConversationService) List(ctx context.Context, f ports.ListConversationsFilter) ([]domain.Conversation, int, error) {
	return s.store.Repos().Conversations.List(ctx, f)
}

// Get loads a conversation with its full history.
func (s *ConversationService) Get(ctx context.Context, id uuid.UUID) (*ConversationDetail, error) {
	conv, err := s.store.Repos().Conversations.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return s.detail(ctx, conv)
}

// GetByVisitorToken loads a conversation for the tokenized visitor view.
func (s *ConversationService) GetByVisitorToken(ctx context.Context, token string) (*ConversationDetail, error) {
	conv, err := s.store.Repos().Conversations.GetByVisitorToken(ctx, token)
	if err != nil {
		return nil, err
	}
	return s.detail(ctx, conv)
}

func (s *ConversationService) detail(ctx context.Context, conv *domain.Conversation) (*ConversationDetail, error) {
	messages, err := s.store.Repos().Messages.ListByConversation(ctx, conv.ID)
	if err != nil {
		return nil, err
	}
	attachments, err := s.store.Repos().Attachments.ListByConversation(ctx, conv.ID)
	if err != nil {
		return nil, err
	}
	return &ConversationDetail{Conversation: *conv, Messages: messages, Attachments: attachments}, nil
}

// Reply adds an outbound message from the host and enqueues delivery.
func (s *ConversationService) Reply(ctx context.Context, conversationID uuid.UUID, body string, attachmentIDs []uuid.UUID) (*domain.Message, error) {
	body = strings.TrimSpace(body)

	v := NewValidate()
	v.Required("body", body)
	v.MaxLen("body", body, 20000)
	v.MaxItems("attachments", len(attachmentIDs), s.maxAttachments)
	if err := v.Err(); err != nil {
		return nil, err
	}

	now := s.clock.Now()
	var message *domain.Message

	err := s.store.WithinTx(ctx, func(repos ports.Repos) error {
		conv, err := repos.Conversations.GetByID(ctx, conversationID)
		if err != nil {
			return err
		}
		message = &domain.Message{
			ID:             s.ids.New(),
			ConversationID: conv.ID,
			Direction:      domain.DirectionOutbound,
			Body:           body,
			FromAddr:       s.hostEmail,
			ToAddr:         conv.VisitorEmail,
			CreatedAt:      now,
		}
		if err := repos.Messages.Create(ctx, message); err != nil {
			return err
		}
		if len(attachmentIDs) > 0 {
			if err := repos.Attachments.LinkToMessage(ctx, attachmentIDs, conv.ID, message.ID); err != nil {
				return err
			}
		}
		conv.Unread = false
		conv.LastActivityAt = now
		if err := repos.Conversations.Update(ctx, conv); err != nil {
			return err
		}
		if err := enqueue(ctx, s.ids, repos, domain.JobSendVisitorMail, SendVisitorMailPayload{
			ConversationID: conv.ID, MessageID: message.ID,
		}); err != nil {
			return err
		}
		for _, id := range attachmentIDs {
			if err := enqueue(ctx, s.ids, repos, domain.JobScanAttachment, ScanAttachmentPayload{AttachmentID: id}); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return message, nil
}

// SetRead updates the unread flag for a conversation.
func (s *ConversationService) SetRead(ctx context.Context, id uuid.UUID, read bool) error {
	conv, err := s.store.Repos().Conversations.GetByID(ctx, id)
	if err != nil {
		return err
	}
	conv.Unread = !read
	return s.store.Repos().Conversations.Update(ctx, conv)
}

// SetArchived archives or restores a conversation.
func (s *ConversationService) SetArchived(ctx context.Context, id uuid.UUID, archived bool) error {
	conv, err := s.store.Repos().Conversations.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if archived {
		conv.Status = domain.StatusArchived
	} else {
		conv.Status = domain.StatusOpen
	}
	return s.store.Repos().Conversations.Update(ctx, conv)
}

// Delete removes a conversation and (via cascade) its messages and attachments.
func (s *ConversationService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.store.Repos().Conversations.Delete(ctx, id)
}
