package app

import (
	"context"
	"errors"
	"testing"

	"github.com/youmei295/something-something/src/backend/internal/domain"
	"github.com/youmei295/something-something/src/backend/internal/platform/clock"
	"github.com/youmei295/something-something/src/backend/internal/platform/id"
)

func newSubmissionService(store *fakeStore) *SubmissionService {
	ids := id.NewGenerator()
	return NewSubmissionService(store, ids, ids, clock.New(), 5)
}

func TestSubmissionService_Submit_HappyPath(t *testing.T) {
	store := newFakeStore()
	svc := newSubmissionService(store)

	result, err := svc.Submit(context.Background(), SubmitInput{
		Name:    "Alice",
		Email:   "alice@example.com",
		Subject: "Hello",
		Body:    "World",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	conv, ok := store.conversations[result.ConversationID]
	if !ok {
		t.Fatal("conversation was not stored")
	}
	if conv.VisitorEmail != "alice@example.com" || conv.Subject != "Hello" {
		t.Fatalf("unexpected conversation: %+v", conv)
	}
	if conv.VisitorToken != result.VisitorToken || conv.VisitorToken == "" {
		t.Fatal("visitor token missing or mismatched")
	}
	if !conv.Unread {
		t.Fatal("new conversation should be unread")
	}

	var inbound int
	for _, m := range store.messages {
		if m.ConversationID == conv.ID && m.Direction == domain.DirectionInbound {
			inbound++
		}
	}
	if inbound != 1 {
		t.Fatalf("expected 1 inbound message, got %d", inbound)
	}

	if got := store.jobTypes()[domain.JobNotifyHost]; got != 1 {
		t.Fatalf("expected 1 notify job, got %d", got)
	}
}

func TestSubmissionService_Submit_Validation(t *testing.T) {
	store := newFakeStore()
	svc := newSubmissionService(store)

	_, err := svc.Submit(context.Background(), SubmitInput{Name: "Alice", Email: "not-an-email"})
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("expected ErrValidation, got %v", err)
	}
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatal("expected *ValidationError")
	}
	for _, field := range []string{"email", "subject", "body"} {
		if _, ok := ve.Fields[field]; !ok {
			t.Fatalf("expected field error for %q, got %+v", field, ve.Fields)
		}
	}
	if len(store.conversations) != 0 {
		t.Fatal("no conversation should be stored on validation failure")
	}
}

func newConversationService(store *fakeStore) *ConversationService {
	return NewConversationService(store, id.NewGenerator(), clock.New(), "host@example.com", 5)
}

func TestConversationService_Reply(t *testing.T) {
	store := newFakeStore()
	sub := newSubmissionService(store)
	convs := newConversationService(store)

	result, err := sub.Submit(context.Background(), SubmitInput{
		Name: "Alice", Email: "alice@example.com", Subject: "Hi", Body: "Question?",
	})
	if err != nil {
		t.Fatalf("submit: %v", err)
	}

	message, err := convs.Reply(context.Background(), result.ConversationID, "Answer!", nil)
	if err != nil {
		t.Fatalf("reply: %v", err)
	}
	if message.Direction != domain.DirectionOutbound {
		t.Fatalf("expected outbound, got %s", message.Direction)
	}
	if message.FromAddr != "host@example.com" || message.ToAddr != "alice@example.com" {
		t.Fatalf("unexpected addresses: %+v", message)
	}

	conv := store.conversations[result.ConversationID]
	if conv.Unread {
		t.Fatal("conversation should be read after replying")
	}
	if got := store.jobTypes()[domain.JobSendVisitorMail]; got != 1 {
		t.Fatalf("expected 1 send mail job, got %d", got)
	}
}

func TestConversationService_Reply_RequiresBody(t *testing.T) {
	store := newFakeStore()
	sub := newSubmissionService(store)
	convs := newConversationService(store)

	result, _ := sub.Submit(context.Background(), SubmitInput{
		Name: "Alice", Email: "alice@example.com", Subject: "Hi", Body: "Question?",
	})

	_, err := convs.Reply(context.Background(), result.ConversationID, "   ", nil)
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("expected ErrValidation, got %v", err)
	}
}
