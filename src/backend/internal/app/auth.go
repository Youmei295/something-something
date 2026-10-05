package app

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/youmei295/something-something/src/backend/internal/domain"
	"github.com/youmei295/something-something/src/backend/internal/ports"
)

// AuthService handles host login sessions.
type AuthService struct {
	store      ports.Store
	hasher     ports.PasswordHasher
	clock      ports.Clock
	ids        ports.IDGenerator
	sessionTTL time.Duration
}

func NewAuthService(store ports.Store, hasher ports.PasswordHasher, clock ports.Clock, ids ports.IDGenerator, sessionTTL time.Duration) *AuthService {
	return &AuthService{store: store, hasher: hasher, clock: clock, ids: ids, sessionTTL: sessionTTL}
}

// LoginResult is returned by Login and carries the session used to set a cookie.
type LoginResult struct {
	User      *domain.User
	Session   *domain.Session
	ExpiresAt time.Time
}

// Login verifies credentials and creates a session. It returns ErrUnauthorized
// for both unknown users and bad passwords to avoid leaking which is which.
func (s *AuthService) Login(ctx context.Context, email, password string) (*LoginResult, error) {
	v := NewValidate()
	v.Required("email", email)
	v.Required("password", password)
	if err := v.Err(); err != nil {
		return nil, err
	}

	user, err := s.store.Repos().Users.GetByEmail(ctx, strings.ToLower(strings.TrimSpace(email)))
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.ErrUnauthorized
		}
		return nil, err
	}

	ok, err := s.hasher.Verify(user.PasswordHash, password)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, domain.ErrUnauthorized
	}

	now := s.clock.Now()
	session := &domain.Session{
		ID:        s.ids.New(),
		UserID:    user.ID,
		ExpiresAt: now.Add(s.sessionTTL),
		CreatedAt: now,
	}
	if err := s.store.Repos().Sessions.Create(ctx, session); err != nil {
		return nil, err
	}
	return &LoginResult{User: user, Session: session, ExpiresAt: session.ExpiresAt}, nil
}

// Authenticate resolves a session ID to a user, enforcing expiry.
func (s *AuthService) Authenticate(ctx context.Context, sessionID uuid.UUID) (*domain.User, error) {
	session, err := s.store.Repos().Sessions.GetByID(ctx, sessionID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.ErrUnauthorized
		}
		return nil, err
	}
	if !session.ExpiresAt.After(s.clock.Now()) {
		_ = s.store.Repos().Sessions.Delete(ctx, session.ID)
		return nil, domain.ErrUnauthorized
	}
	return s.store.Repos().Users.GetByID(ctx, session.UserID)
}

// Logout deletes the session.
func (s *AuthService) Logout(ctx context.Context, sessionID uuid.UUID) error {
	return s.store.Repos().Sessions.Delete(ctx, sessionID)
}

// SeedHost creates the host account if no users exist yet. It is idempotent and
// safe to run on every startup.
func (s *AuthService) SeedHost(ctx context.Context, email, name, password string) (bool, error) {
	n, err := s.store.Repos().Users.Count(ctx)
	if err != nil {
		return false, err
	}
	if n > 0 {
		return false, nil
	}
	if strings.TrimSpace(password) == "" {
		return false, errors.New("seed host: HOST_PASSWORD must be set when no users exist")
	}
	hash, err := s.hasher.Hash(password)
	if err != nil {
		return false, err
	}
	now := s.clock.Now()
	user := &domain.User{
		ID:           s.ids.New(),
		Email:        strings.ToLower(strings.TrimSpace(email)),
		PasswordHash: hash,
		Role:         "host",
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := s.store.Repos().Users.Create(ctx, user); err != nil {
		return false, err
	}
	return true, nil
}
