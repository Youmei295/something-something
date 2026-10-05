package ports

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// ObjectInfo is storage-agnostic metadata about a stored object.
type ObjectInfo struct {
	Size        int64
	ContentType string
	ETag        string
}

// Storage abstracts object storage (MinIO locally, S3 in production). All
// methods work against an S3-compatible API so the adapter can be swapped.
type Storage interface {
	EnsureBucket(ctx context.Context) error
	PresignPut(ctx context.Context, key, contentType string, ttl time.Duration) (string, error)
	PresignGet(ctx context.Context, key, downloadFilename, contentType string, ttl time.Duration) (string, error)
	Stat(ctx context.Context, key string) (ObjectInfo, error)
	Remove(ctx context.Context, key string) error
}

// Email is a transport-agnostic outbound message.
type Email struct {
	To       string
	ToName   string
	Subject  string
	TextBody string
	HTMLBody string
	ReplyTo  string
	Headers  map[string]string
}

// Mailer delivers outbound email. Implementations include console (dev), SMTP,
// and a transactional provider (Postmark). Swapping providers is a wiring change.
type Mailer interface {
	Send(ctx context.Context, e Email) error
}

// PasswordHasher hashes and verifies host passwords.
type PasswordHasher interface {
	Hash(plain string) (string, error)
	Verify(hash, plain string) (bool, error)
}

// IDGenerator produces new identifiers, allowing deterministic fakes in tests.
type IDGenerator interface {
	New() uuid.UUID
}

// TokenGenerator produces cryptographically random opaque tokens for visitor
// thread links.
type TokenGenerator interface {
	NewToken(n int) (string, error)
}

// Clock abstracts time so use cases can be tested without sleeping.
type Clock interface {
	Now() time.Time
}

// VisitorLinkBuilder renders the tokenized visitor thread URL.
type VisitorLinkBuilder interface {
	ThreadURL(token string) string
}
