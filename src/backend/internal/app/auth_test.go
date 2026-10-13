package app

import (
	"context"
	"testing"
	"time"

	"github.com/youmei295/something-something/src/backend/internal/adapters/auth"
	"github.com/youmei295/something-something/src/backend/internal/platform/clock"
	"github.com/youmei295/something-something/src/backend/internal/platform/id"
)

func newAuthService(store *fakeStore) *AuthService {
	return NewAuthService(store, auth.NewArgon2Hasher(), clock.New(), id.NewGenerator(), time.Hour)
}

func TestSeedHost_EnsuresExampleAccount(t *testing.T) {
	store := newFakeStore()
	svc := newAuthService(store)
	ctx := context.Background()

	created, err := svc.SeedHost(ctx, "host@example.com", "Host", "change-me-too")
	if err != nil || !created {
		t.Fatalf("first seed: created=%v err=%v", created, err)
	}

	// Idempotent: running again must not create or reset anything.
	created, err = svc.SeedHost(ctx, "host@example.com", "Host", "a-different-password")
	if err != nil || created {
		t.Fatalf("second seed should be a no-op: created=%v err=%v", created, err)
	}

	// The original password still works (not overwritten).
	result, err := svc.Login(ctx, "host@example.com", "change-me-too")
	if err != nil {
		t.Fatalf("login with seeded password: %v", err)
	}
	if result.User.Email != "host@example.com" || result.User.Role != "host" {
		t.Fatalf("unexpected user: %+v", result.User)
	}

	// A different host email is created as well.
	created, err = svc.SeedHost(ctx, "second@example.com", "Second", "pw")
	if err != nil || !created {
		t.Fatalf("seed second host: created=%v err=%v", created, err)
	}
}

func TestSeedHost_RequiresEmailAndPassword(t *testing.T) {
	store := newFakeStore()
	svc := newAuthService(store)
	ctx := context.Background()

	if _, err := svc.SeedHost(ctx, "", "Host", "pw"); err == nil {
		t.Fatal("expected error for empty email")
	}
	if _, err := svc.SeedHost(ctx, "host@example.com", "Host", ""); err == nil {
		t.Fatal("expected error for empty password")
	}
}
