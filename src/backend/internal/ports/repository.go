// Package ports defines the interfaces (driven and driving) that the
// application core depends on. Adapters in internal/adapters implement them and
// the application services consume them, keeping both sides replaceable.
package ports

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/youmei295/something-something/src/backend/internal/domain"
)

// Repos bundles the repositories available inside a single unit of work. The
// concrete adapter decides whether they are bound to a pool or a transaction.
type Repos struct {
	Users         UserRepository
	Sessions      SessionRepository
	Conversations ConversationRepository
	Messages      MessageRepository
	Attachments   AttachmentRepository
	Jobs          JobRepository
}

// Store opens units of work and exposes the default (non-transactional) repos.
type Store interface {
	Repos() Repos
	WithinTx(ctx context.Context, fn func(Repos) error) error
	Ping(ctx context.Context) error
	Close()
}

// ListConversationsFilter describes how the inbox list is queried.
type ListConversationsFilter struct {
	Status ConversationStatusFilter
	Search string
	Limit  int
	Offset int
}

// ConversationStatusFilter limits listing to open or archived threads.
type ConversationStatusFilter string

const (
	FilterAll      ConversationStatusFilter = "all"
	FilterOpen     ConversationStatusFilter = "open"
	FilterArchived ConversationStatusFilter = "archived"
)

type UserRepository interface {
	Create(ctx context.Context, u *domain.User) error
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
	Count(ctx context.Context) (int, error)
}

type SessionRepository interface {
	Create(ctx context.Context, s *domain.Session) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Session, error)
	Delete(ctx context.Context, id uuid.UUID) error
	DeleteExpired(ctx context.Context, before time.Time) (int64, error)
}

type ConversationRepository interface {
	Create(ctx context.Context, c *domain.Conversation) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Conversation, error)
	GetByVisitorToken(ctx context.Context, token string) (*domain.Conversation, error)
	List(ctx context.Context, f ListConversationsFilter) ([]domain.Conversation, int, error)
	Update(ctx context.Context, c *domain.Conversation) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type MessageRepository interface {
	Create(ctx context.Context, m *domain.Message) error
	ListByConversation(ctx context.Context, conversationID uuid.UUID) ([]domain.Message, error)
	GetByEmailMessageID(ctx context.Context, emailMessageID string) (*domain.Message, error)
}

type AttachmentRepository interface {
	Create(ctx context.Context, a *domain.Attachment) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Attachment, error)
	ListByConversation(ctx context.Context, conversationID uuid.UUID) ([]domain.Attachment, error)
	LinkToMessage(ctx context.Context, ids []uuid.UUID, conversationID, messageID uuid.UUID) error
	MarkUploaded(ctx context.Context, id uuid.UUID, size int64) error
	MarkScanStatus(ctx context.Context, id uuid.UUID, status domain.ScanStatus) error
	ListOrphans(ctx context.Context, before time.Time) ([]domain.Attachment, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type JobRepository interface {
	Enqueue(ctx context.Context, j *domain.Job) error
	// Dequeue atomically claims up to limit due jobs of the given types and marks
	// them running, so multiple workers can run safely in parallel.
	Dequeue(ctx context.Context, types []string, limit int) ([]domain.Job, error)
	Complete(ctx context.Context, id uuid.UUID) error
	Fail(ctx context.Context, id uuid.UUID, errMsg string, retryAt time.Time) error
}
