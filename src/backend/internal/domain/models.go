// Package domain contains the core business entities and value types.
//
// It deliberately has no dependencies on HTTP, SQL, or any external SDK so the
// rules of the product stay independent from the infrastructure that serves them.
package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// Sentinel errors returned by repositories and services. Transport layers map
// these to HTTP status codes, so they form a stable contract across adapters.
var (
	ErrNotFound      = errors.New("not found")
	ErrConflict      = errors.New("conflict")
	ErrUnauthorized  = errors.New("unauthorized")
	ErrForbidden     = errors.New("forbidden")
	ErrValidation    = errors.New("validation failed")
	ErrPayloadTooBig = errors.New("payload too large")
	ErrUnsupported   = errors.New("unsupported media type")
)

// Direction marks whether a message came from a visitor or the host.
type Direction string

const (
	DirectionInbound  Direction = "inbound"
	DirectionOutbound Direction = "outbound"
)

// ConversationStatus controls whether a thread appears in the active inbox.
type ConversationStatus string

const (
	StatusOpen     ConversationStatus = "open"
	StatusArchived ConversationStatus = "archived"
)

// ScanStatus tracks malware scanning of an attachment.
type ScanStatus string

const (
	ScanPending  ScanStatus = "pending"
	ScanClean    ScanStatus = "clean"
	ScanInfected ScanStatus = "infected"
	ScanFailed   ScanStatus = "failed"
)

// UploadStatus tracks the lifecycle of an attachment between requesting a
// presigned URL and being linked to a message.
type UploadStatus string

const (
	UploadPending  UploadStatus = "pending"
	UploadComplete UploadStatus = "complete"
	UploadLinked   UploadStatus = "linked"
)

// User is a host account able to read and reply to conversations.
type User struct {
	ID           uuid.UUID
	Email        string
	PasswordHash string
	Role         string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// Session is a server-side host login session referenced by an httpOnly cookie.
type Session struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	ExpiresAt time.Time
	CreatedAt time.Time
}

// Conversation groups messages exchanged between a single visitor and the host.
type Conversation struct {
	ID             uuid.UUID
	Subject        string
	Status         ConversationStatus
	VisitorName    string
	VisitorEmail   string
	VisitorToken   string
	Unread         bool
	CreatedAt      time.Time
	LastActivityAt time.Time
}

// Message is a single inbound or outbound item within a conversation.
type Message struct {
	ID             uuid.UUID
	ConversationID uuid.UUID
	Direction      Direction
	Body           string
	FromAddr       string
	ToAddr         string
	EmailMessageID string
	InReplyTo      string
	CreatedAt      time.Time
}

// Attachment is a file uploaded by a visitor or host and linked to a message.
// ConversationID and MessageID are nil until the upload is linked.
type Attachment struct {
	ID             uuid.UUID
	ConversationID *uuid.UUID
	MessageID      *uuid.UUID
	Filename       string
	ContentType    string
	Size           int64
	StorageKey     string
	SHA256         string
	UploadStatus   UploadStatus
	ScanStatus     ScanStatus
	CreatedAt      time.Time
}

// IsDownloadable reports whether an attachment may be served to a client.
func (a Attachment) IsDownloadable() bool {
	return a.UploadStatus == UploadLinked && a.ScanStatus == ScanClean
}

// JobStatus describes the state of a background job.
type JobStatus string

const (
	JobPending JobStatus = "pending"
	JobRunning JobStatus = "running"
	JobDone    JobStatus = "done"
	JobFailed  JobStatus = "failed"
)

// Job is a durable unit of asynchronous work processed by the worker.
type Job struct {
	ID          uuid.UUID
	Type        string
	Payload     []byte
	Status      JobStatus
	Attempts    int
	MaxAttempts int
	RunAt       time.Time
	LastError   string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Known job types. Keeping them as constants avoids typos across producer and
// consumer and makes the set of side effects discoverable.
const (
	JobNotifyHost      = "notify_host"
	JobSendVisitorMail = "send_visitor_mail"
	JobScanAttachment  = "scan_attachment"
	JobCleanupUploads  = "cleanup_uploads"
)
