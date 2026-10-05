package app

import (
	"context"
	"sort"
	"time"

	"github.com/google/uuid"

	"github.com/youmei295/something-something/src/backend/internal/domain"
	"github.com/youmei295/something-something/src/backend/internal/ports"
)

// fakeStore is an in-memory ports.Store used to test use cases without a
// database. Its existence is a direct benefit of depending on interfaces.

type fakeStore struct {
	users         map[uuid.UUID]*domain.User
	sessions      map[uuid.UUID]*domain.Session
	conversations map[uuid.UUID]*domain.Conversation
	messages      map[uuid.UUID]*domain.Message
	attachments   map[uuid.UUID]*domain.Attachment
	jobs          []*domain.Job
}

func newFakeStore() *fakeStore {
	return &fakeStore{
		users:         map[uuid.UUID]*domain.User{},
		sessions:      map[uuid.UUID]*domain.Session{},
		conversations: map[uuid.UUID]*domain.Conversation{},
		messages:      map[uuid.UUID]*domain.Message{},
		attachments:   map[uuid.UUID]*domain.Attachment{},
	}
}

func (s *fakeStore) Repos() ports.Repos { return s.repos() }

func (s *fakeStore) repos() ports.Repos {
	return ports.Repos{
		Users:         &fakeUsers{s},
		Sessions:      &fakeSessions{s},
		Conversations: &fakeConversations{s},
		Messages:      &fakeMessages{s},
		Attachments:   &fakeAttachments{s},
		Jobs:          &fakeJobs{s},
	}
}

func (s *fakeStore) WithinTx(_ context.Context, fn func(ports.Repos) error) error {
	return fn(s.repos())
}
func (s *fakeStore) Ping(context.Context) error { return nil }
func (s *fakeStore) Close()                     {}

// ---- users ----

type fakeUsers struct{ s *fakeStore }

func (r *fakeUsers) Create(_ context.Context, u *domain.User) error { r.s.users[u.ID] = u; return nil }
func (r *fakeUsers) GetByEmail(_ context.Context, email string) (*domain.User, error) {
	for _, u := range r.s.users {
		if u.Email == email {
			return u, nil
		}
	}
	return nil, domain.ErrNotFound
}
func (r *fakeUsers) GetByID(_ context.Context, id uuid.UUID) (*domain.User, error) {
	if u, ok := r.s.users[id]; ok {
		return u, nil
	}
	return nil, domain.ErrNotFound
}
func (r *fakeUsers) Count(context.Context) (int, error) { return len(r.s.users), nil }

// ---- sessions ----

type fakeSessions struct{ s *fakeStore }

func (r *fakeSessions) Create(_ context.Context, s *domain.Session) error {
	r.s.sessions[s.ID] = s
	return nil
}
func (r *fakeSessions) GetByID(_ context.Context, id uuid.UUID) (*domain.Session, error) {
	if s, ok := r.s.sessions[id]; ok {
		return s, nil
	}
	return nil, domain.ErrNotFound
}
func (r *fakeSessions) Delete(_ context.Context, id uuid.UUID) error {
	delete(r.s.sessions, id)
	return nil
}
func (r *fakeSessions) DeleteExpired(_ context.Context, before time.Time) (int64, error) {
	var n int64
	for id, s := range r.s.sessions {
		if s.ExpiresAt.Before(before) {
			delete(r.s.sessions, id)
			n++
		}
	}
	return n, nil
}

// ---- conversations ----

type fakeConversations struct{ s *fakeStore }

func (r *fakeConversations) Create(_ context.Context, c *domain.Conversation) error {
	r.s.conversations[c.ID] = c
	return nil
}
func (r *fakeConversations) GetByID(_ context.Context, id uuid.UUID) (*domain.Conversation, error) {
	if c, ok := r.s.conversations[id]; ok {
		return c, nil
	}
	return nil, domain.ErrNotFound
}
func (r *fakeConversations) GetByVisitorToken(_ context.Context, token string) (*domain.Conversation, error) {
	for _, c := range r.s.conversations {
		if c.VisitorToken == token {
			return c, nil
		}
	}
	return nil, domain.ErrNotFound
}
func (r *fakeConversations) List(_ context.Context, f ports.ListConversationsFilter) ([]domain.Conversation, int, error) {
	out := make([]domain.Conversation, 0, len(r.s.conversations))
	for _, c := range r.s.conversations {
		if f.Status != "" && f.Status != ports.FilterAll && string(c.Status) != string(f.Status) {
			continue
		}
		out = append(out, *c)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].LastActivityAt.After(out[j].LastActivityAt) })
	total := len(out)
	return out, total, nil
}
func (r *fakeConversations) Update(_ context.Context, c *domain.Conversation) error {
	if _, ok := r.s.conversations[c.ID]; !ok {
		return domain.ErrNotFound
	}
	r.s.conversations[c.ID] = c
	return nil
}
func (r *fakeConversations) Delete(_ context.Context, id uuid.UUID) error {
	delete(r.s.conversations, id)
	return nil
}

// ---- messages ----

type fakeMessages struct{ s *fakeStore }

func (r *fakeMessages) Create(_ context.Context, m *domain.Message) error {
	r.s.messages[m.ID] = m
	return nil
}
func (r *fakeMessages) ListByConversation(_ context.Context, conversationID uuid.UUID) ([]domain.Message, error) {
	out := make([]domain.Message, 0)
	for _, m := range r.s.messages {
		if m.ConversationID == conversationID {
			out = append(out, *m)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out, nil
}
func (r *fakeMessages) GetByEmailMessageID(_ context.Context, emailMessageID string) (*domain.Message, error) {
	for _, m := range r.s.messages {
		if m.EmailMessageID == emailMessageID {
			return m, nil
		}
	}
	return nil, domain.ErrNotFound
}

// ---- attachments ----

type fakeAttachments struct{ s *fakeStore }

func (r *fakeAttachments) Create(_ context.Context, a *domain.Attachment) error {
	r.s.attachments[a.ID] = a
	return nil
}
func (r *fakeAttachments) GetByID(_ context.Context, id uuid.UUID) (*domain.Attachment, error) {
	if a, ok := r.s.attachments[id]; ok {
		return a, nil
	}
	return nil, domain.ErrNotFound
}
func (r *fakeAttachments) ListByConversation(_ context.Context, conversationID uuid.UUID) ([]domain.Attachment, error) {
	out := make([]domain.Attachment, 0)
	for _, a := range r.s.attachments {
		if a.ConversationID != nil && *a.ConversationID == conversationID {
			out = append(out, *a)
		}
	}
	return out, nil
}
func (r *fakeAttachments) LinkToMessage(_ context.Context, ids []uuid.UUID, conversationID, messageID uuid.UUID) error {
	for _, id := range ids {
		if a, ok := r.s.attachments[id]; ok {
			conv, msg := conversationID, messageID
			a.ConversationID = &conv
			a.MessageID = &msg
			a.UploadStatus = domain.UploadLinked
		}
	}
	return nil
}
func (r *fakeAttachments) MarkUploaded(_ context.Context, id uuid.UUID, size int64) error {
	a, ok := r.s.attachments[id]
	if !ok {
		return domain.ErrNotFound
	}
	a.UploadStatus = domain.UploadComplete
	a.Size = size
	return nil
}
func (r *fakeAttachments) MarkScanStatus(_ context.Context, id uuid.UUID, status domain.ScanStatus) error {
	if a, ok := r.s.attachments[id]; ok {
		a.ScanStatus = status
	}
	return nil
}
func (r *fakeAttachments) ListOrphans(_ context.Context, before time.Time) ([]domain.Attachment, error) {
	out := make([]domain.Attachment, 0)
	for _, a := range r.s.attachments {
		if a.CreatedAt.Before(before) && a.UploadStatus != domain.UploadLinked {
			out = append(out, *a)
		}
	}
	return out, nil
}
func (r *fakeAttachments) Delete(_ context.Context, id uuid.UUID) error {
	delete(r.s.attachments, id)
	return nil
}

// ---- jobs ----

type fakeJobs struct{ s *fakeStore }

func (r *fakeJobs) Enqueue(_ context.Context, j *domain.Job) error {
	cp := *j
	r.s.jobs = append(r.s.jobs, &cp)
	return nil
}
func (r *fakeJobs) Dequeue(_ context.Context, types []string, limit int) ([]domain.Job, error) {
	out := make([]domain.Job, 0)
	for _, j := range r.s.jobs {
		if j.Status != domain.JobPending {
			continue
		}
		if len(types) > 0 && !contains(types, j.Type) {
			continue
		}
		j.Status = domain.JobRunning
		j.Attempts++
		out = append(out, *j)
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}
func (r *fakeJobs) Complete(_ context.Context, id uuid.UUID) error {
	for _, j := range r.s.jobs {
		if j.ID == id {
			j.Status = domain.JobDone
		}
	}
	return nil
}
func (r *fakeJobs) Fail(_ context.Context, id uuid.UUID, errMsg string, retryAt time.Time) error {
	for _, j := range r.s.jobs {
		if j.ID == id {
			j.LastError = errMsg
			j.RunAt = retryAt
			if j.Attempts >= j.MaxAttempts {
				j.Status = domain.JobFailed
			} else {
				j.Status = domain.JobPending
			}
		}
	}
	return nil
}

func (s *fakeStore) jobTypes() map[string]int {
	out := map[string]int{}
	for _, j := range s.jobs {
		out[j.Type]++
	}
	return out
}

func contains(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}
