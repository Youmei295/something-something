package app

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/youmei295/something-something/src/backend/internal/domain"
	"github.com/youmei295/something-something/src/backend/internal/ports"
)

// Worker consumes background jobs (notifications, outbound mail, scanning,
// cleanup). It is intentionally transport-agnostic: swapping the job table for
// an event bus later only changes the dequeue/ack implementation.
type Worker struct {
	store        ports.Store
	mailer       ports.Mailer
	ids          ports.IDGenerator
	clock        ports.Clock
	log          *slog.Logger
	links        ports.VisitorLinkBuilder
	hostEmail    string
	hostName     string
	pollInterval time.Duration
	concurrency  int
}

func NewWorker(store ports.Store, mailer ports.Mailer, ids ports.IDGenerator, clock ports.Clock, log *slog.Logger, links ports.VisitorLinkBuilder, hostEmail, hostName string, pollInterval time.Duration, concurrency int) *Worker {
	return &Worker{
		store: store, mailer: mailer, ids: ids, clock: clock, log: log, links: links,
		hostEmail: hostEmail, hostName: hostName, pollInterval: pollInterval, concurrency: concurrency,
	}
}

// Run polls for jobs until the context is cancelled.
func (w *Worker) Run(ctx context.Context) error {
	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	w.log.Info("worker started", slog.Duration("poll_interval", w.pollInterval))
	for {
		select {
		case <-ctx.Done():
			w.log.Info("worker stopping")
			return ctx.Err()
		case <-ticker.C:
			if err := w.processBatch(ctx); err != nil {
				w.log.Error("worker batch failed", slog.String("error", err.Error()))
			}
		}
	}
}

func (w *Worker) processBatch(ctx context.Context) error {
	jobs, err := w.store.Repos().Jobs.Dequeue(ctx, nil, w.concurrency)
	if err != nil {
		return err
	}
	for _, job := range jobs {
		w.handle(ctx, job)
	}
	return nil
}

func (w *Worker) handle(ctx context.Context, job domain.Job) {
	var err error
	switch job.Type {
	case domain.JobNotifyHost:
		err = w.notifyHost(ctx, job)
	case domain.JobSendVisitorMail:
		err = w.sendVisitorMail(ctx, job)
	case domain.JobScanAttachment:
		err = w.scanAttachment(ctx, job)
	case domain.JobCleanupUploads:
		err = w.cleanupUploads(ctx, job)
	default:
		err = fmt.Errorf("unknown job type %q", job.Type)
	}

	if err == nil {
		if e := w.store.Repos().Jobs.Complete(ctx, job.ID); e != nil {
			w.log.Error("complete job failed", slog.String("job_id", job.ID.String()), slog.String("error", e.Error()))
		}
		return
	}

	// Exponential backoff capped at 10 minutes.
	backoff := time.Duration(1<<min(job.Attempts, 9)) * time.Second
	retryAt := w.clock.Now().Add(backoff)
	if e := w.store.Repos().Jobs.Fail(ctx, job.ID, err.Error(), retryAt); e != nil {
		w.log.Error("fail job failed", slog.String("job_id", job.ID.String()), slog.String("error", e.Error()))
	}
	w.log.Warn("job failed",
		slog.String("job_id", job.ID.String()),
		slog.String("type", job.Type),
		slog.Int("attempts", job.Attempts),
		slog.String("error", err.Error()),
	)
}

func (w *Worker) notifyHost(ctx context.Context, job domain.Job) error {
	var payload NotifyHostPayload
	if err := json.Unmarshal(job.Payload, &payload); err != nil {
		return err
	}
	conversation, message, err := w.load(ctx, payload.ConversationID, payload.MessageID)
	if err != nil {
		return err
	}
	link := w.links.ThreadURL(conversation.VisitorToken)
	return w.mailer.Send(ctx, ports.Email{
		To:      w.hostEmail,
		ToName:  w.hostName,
		Subject: "New message: " + conversation.Subject,
		TextBody: fmt.Sprintf("%s <%s> wrote:\n\n%s\n\nReply: %s\n",
			conversation.VisitorName, conversation.VisitorEmail, message.Body, link),
	})
}

func (w *Worker) sendVisitorMail(ctx context.Context, job domain.Job) error {
	var payload SendVisitorMailPayload
	if err := json.Unmarshal(job.Payload, &payload); err != nil {
		return err
	}
	conversation, message, err := w.load(ctx, payload.ConversationID, payload.MessageID)
	if err != nil {
		return err
	}
	link := w.links.ThreadURL(conversation.VisitorToken)
	return w.mailer.Send(ctx, ports.Email{
		To:       conversation.VisitorEmail,
		ToName:   conversation.VisitorName,
		Subject:  "Re: " + conversation.Subject,
		ReplyTo:  w.hostEmail,
		TextBody: fmt.Sprintf("%s\n\n---\nView this conversation: %s\n", message.Body, link),
	})
}

// scanAttachment is a placeholder for malware scanning. Until ClamAV is
// introduced (v0.2), uploads are marked clean so downloads work end to end.
func (w *Worker) scanAttachment(ctx context.Context, job domain.Job) error {
	var payload ScanAttachmentPayload
	if err := json.Unmarshal(job.Payload, &payload); err != nil {
		return err
	}
	return w.store.Repos().Attachments.MarkScanStatus(ctx, payload.AttachmentID, domain.ScanClean)
}

func (w *Worker) cleanupUploads(ctx context.Context, job domain.Job) error {
	var payload CleanupUploadsPayload
	if err := json.Unmarshal(job.Payload, &payload); err != nil {
		return err
	}
	olderThan := payload.OlderThan
	if olderThan <= 0 {
		olderThan = 24 * time.Hour
	}
	cutoff := w.clock.Now().Add(-olderThan)
	orphans, err := w.store.Repos().Attachments.ListOrphans(ctx, cutoff)
	if err != nil {
		return err
	}
	for _, a := range orphans {
		_ = w.store.Repos().Attachments.Delete(ctx, a.ID)
	}
	return nil
}

func (w *Worker) load(ctx context.Context, conversationID, messageID uuid.UUID) (*domain.Conversation, *domain.Message, error) {
	conversation, err := w.store.Repos().Conversations.GetByID(ctx, conversationID)
	if err != nil {
		return nil, nil, err
	}
	messages, err := w.store.Repos().Messages.ListByConversation(ctx, conversationID)
	if err != nil {
		return nil, nil, err
	}
	for i := range messages {
		if messages[i].ID == messageID {
			return conversation, &messages[i], nil
		}
	}
	return nil, nil, domain.ErrNotFound
}
