package app

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/youmei295/something-something/src/backend/internal/domain"
	"github.com/youmei295/something-something/src/backend/internal/ports"
)

// AttachmentService manages attachment upload and download. It validates files
// before issuing presigned URLs and never exposes storage keys to clients.
type AttachmentService struct {
	store      ports.Store
	storage    ports.Storage
	ids        ports.IDGenerator
	clock      ports.Clock
	maxBytes   int64
	allowed    map[string]struct{}
	presignTTL time.Duration
}

func NewAttachmentService(store ports.Store, storage ports.Storage, ids ports.IDGenerator, clock ports.Clock, maxBytes int64, allowed []string, presignTTL time.Duration) *AttachmentService {
	allowedSet := make(map[string]struct{}, len(allowed))
	for _, t := range allowed {
		allowedSet[strings.ToLower(strings.TrimSpace(t))] = struct{}{}
	}
	return &AttachmentService{store: store, storage: storage, ids: ids, clock: clock, maxBytes: maxBytes, allowed: allowedSet, presignTTL: presignTTL}
}

// RequestUploadInput describes a file the client wants to upload.
type RequestUploadInput struct {
	Filename    string
	ContentType string
	Size        int64
}

// UploadTicket is returned to the client to perform the actual upload.
type UploadTicket struct {
	AttachmentID uuid.UUID `json:"attachment_id"`
	UploadURL    string    `json:"upload_url"`
	StorageKey   string    `json:"storage_key"`
	ExpiresIn    int       `json:"expires_in"`
}

// RequestUpload validates the file and returns a presigned PUT URL.
func (s *AttachmentService) RequestUpload(ctx context.Context, in RequestUploadInput) (*UploadTicket, error) {
	in.Filename = strings.TrimSpace(in.Filename)
	in.ContentType = strings.ToLower(strings.TrimSpace(in.ContentType))

	v := NewValidate()
	v.Required("filename", in.Filename)
	v.MaxLen("filename", in.Filename, 255)
	v.Required("content_type", in.ContentType)
	if in.Size <= 0 {
		v.Add("size", "must be greater than zero")
	}
	if in.Size > s.maxBytes {
		v.Add("size", fmt.Sprintf("must be at most %d bytes", s.maxBytes))
	}
	if err := v.Err(); err != nil {
		return nil, err
	}
	if _, ok := s.allowed[in.ContentType]; !ok {
		return nil, fmt.Errorf("%w: content type %q is not allowed", domain.ErrUnsupported, in.ContentType)
	}

	id := s.ids.New()
	ext := strings.ToLower(filepath.Ext(in.Filename))
	storageKey := fmt.Sprintf("attachments/%s%s", id, ext)

	attachment := &domain.Attachment{
		ID:           id,
		Filename:     in.Filename,
		ContentType:  in.ContentType,
		Size:         in.Size,
		StorageKey:   storageKey,
		UploadStatus: domain.UploadPending,
		ScanStatus:   domain.ScanPending,
		CreatedAt:    s.clock.Now(),
	}
	if err := s.store.Repos().Attachments.Create(ctx, attachment); err != nil {
		return nil, err
	}

	url, err := s.storage.PresignPut(ctx, storageKey, in.ContentType, s.presignTTL)
	if err != nil {
		return nil, err
	}
	return &UploadTicket{
		AttachmentID: id,
		UploadURL:    url,
		StorageKey:   storageKey,
		ExpiresIn:    int(s.presignTTL.Seconds()),
	}, nil
}

// CompleteUpload verifies the object exists and marks the upload complete.
func (s *AttachmentService) CompleteUpload(ctx context.Context, id uuid.UUID) (*domain.Attachment, error) {
	attachment, err := s.store.Repos().Attachments.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if attachment.UploadStatus != domain.UploadPending && attachment.UploadStatus != domain.UploadComplete {
		return nil, domain.ErrConflict
	}

	info, err := s.storage.Stat(ctx, attachment.StorageKey)
	if err != nil {
		return nil, fmt.Errorf("attachment not uploaded: %w", domain.ErrNotFound)
	}
	if info.Size > s.maxBytes {
		_ = s.storage.Remove(ctx, attachment.StorageKey)
		return nil, domain.ErrPayloadTooBig
	}

	if attachment.UploadStatus == domain.UploadPending {
		if err := s.store.Repos().Attachments.MarkUploaded(ctx, id, info.Size); err != nil {
			return nil, err
		}
	}
	attachment.UploadStatus = domain.UploadComplete
	attachment.Size = info.Size
	return attachment, nil
}

// Get returns an attachment by ID.
func (s *AttachmentService) Get(ctx context.Context, id uuid.UUID) (*domain.Attachment, error) {
	return s.store.Repos().Attachments.GetByID(ctx, id)
}

// DownloadURL returns a short-lived presigned URL, but only for attachments that
// are linked and have passed scanning.
func (s *AttachmentService) DownloadURL(ctx context.Context, id uuid.UUID) (string, error) {
	attachment, err := s.store.Repos().Attachments.GetByID(ctx, id)
	if err != nil {
		return "", err
	}
	if attachment.ConversationID == nil {
		return "", domain.ErrNotFound
	}
	if !attachment.IsDownloadable() {
		return "", fmt.Errorf("%w: attachment is not ready for download", domain.ErrForbidden)
	}
	return s.storage.PresignGet(ctx, attachment.StorageKey, attachment.Filename, attachment.ContentType, s.presignTTL)
}

// CleanupOrphans removes uploads that were never linked to a message.
func (s *AttachmentService) CleanupOrphans(ctx context.Context, olderThan time.Duration) (int, error) {
	cutoff := s.clock.Now().Add(-olderThan)
	orphans, err := s.store.Repos().Attachments.ListOrphans(ctx, cutoff)
	if err != nil {
		return 0, err
	}
	removed := 0
	for _, a := range orphans {
		_ = s.storage.Remove(ctx, a.StorageKey)
		if err := s.store.Repos().Attachments.Delete(ctx, a.ID); err != nil {
			continue
		}
		removed++
	}
	return removed, nil
}

// MaxUploadBytes exposes the configured limit to transports for validation.
func (s *AttachmentService) MaxUploadBytes() int64 { return s.maxBytes }
