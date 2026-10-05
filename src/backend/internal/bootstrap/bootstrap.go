// Package bootstrap is the composition root. It builds the shared infrastructure
// (database, storage, mailer) and the application services, and lets each
// service binary opt into only the pieces it needs.
//
// This is what makes the services independently runnable: identity creates no
// storage client, the attachment service creates no mailer, and so on, while
// still sharing one place that knows how to wire dependencies.
package bootstrap

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/youmei295/something-something/src/backend/internal/adapters/auth"
	"github.com/youmei295/something-something/src/backend/internal/adapters/mailer"
	"github.com/youmei295/something-something/src/backend/internal/adapters/postgres"
	"github.com/youmei295/something-something/src/backend/internal/adapters/storage"
	"github.com/youmei295/something-something/src/backend/internal/app"
	"github.com/youmei295/something-something/src/backend/internal/config"
	"github.com/youmei295/something-something/src/backend/internal/platform/clock"
	"github.com/youmei295/something-something/src/backend/internal/platform/id"
	"github.com/youmei295/something-something/src/backend/internal/platform/link"
	"github.com/youmei295/something-something/src/backend/internal/ports"
	transporthttp "github.com/youmei295/something-something/src/backend/internal/transport/http"
)

// Core holds shared infrastructure. Service-specific dependencies (storage,
// mailer, services) are built lazily on first use and cached, so a service only
// pays for what it actually depends on.
type Core struct {
	Config *config.Config
	Logger *slog.Logger
	Store  ports.Store
	IDs    *id.Generator
	Clock  clock.Clock
	Hasher *auth.Argon2Hasher
	Links  ports.VisitorLinkBuilder

	storageOnce sync.Once
	storage     ports.Storage
	storageErr  error

	mailerOnce sync.Once
	mailer     ports.Mailer
	mailerErr  error

	servicesOnce sync.Once
	authSvc      *app.AuthService
}

// NewCore opens the database connection shared by all services.
func NewCore(ctx context.Context, cfg *config.Config, logger *slog.Logger) (*Core, error) {
	store, err := postgres.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("bootstrap: database: %w", err)
	}
	ids := id.NewGenerator()
	return &Core{
		Config: cfg,
		Logger: logger,
		Store:  store,
		IDs:    ids,
		Clock:  clock.New(),
		Hasher: auth.NewArgon2Hasher(),
		Links:  link.NewBuilder(cfg.HTTP.PublicBaseURL),
	}, nil
}

// Storage builds (once) the S3-compatible client and ensures the bucket exists.
func (c *Core) Storage(ctx context.Context) (ports.Storage, error) {
	c.storageOnce.Do(func() {
		objectStorage, err := storage.New(ctx, storage.Options{
			Endpoint:       c.Config.Storage.Endpoint,
			PublicEndpoint: c.Config.Storage.PublicEndpoint,
			AccessKey:      c.Config.Storage.AccessKey,
			SecretKey:      c.Config.Storage.SecretKey,
			Bucket:         c.Config.Storage.Bucket,
			Region:         c.Config.Storage.Region,
			UseSSL:         c.Config.Storage.UseSSL,
		})
		if err != nil {
			c.storageErr = fmt.Errorf("bootstrap: storage: %w", err)
			return
		}
		if err := objectStorage.EnsureBucket(ctx); err != nil {
			c.storageErr = fmt.Errorf("bootstrap: ensure bucket: %w", err)
			return
		}
		c.storage = objectStorage
	})
	return c.storage, c.storageErr
}

// Mailer builds (once) the configured email transport.
func (c *Core) Mailer() (ports.Mailer, error) {
	c.mailerOnce.Do(func() {
		m, err := mailer.New(c.Config.Mailer, c.Logger)
		if err != nil {
			c.mailerErr = fmt.Errorf("bootstrap: mailer: %w", err)
			return
		}
		c.mailer = m
	})
	return c.mailer, c.mailerErr
}

// AuthService is shared by identity (writes sessions) and by any service that
// validates sessions for protected routes.
func (c *Core) AuthService() *app.AuthService {
	c.servicesOnce.Do(func() {
		c.authSvc = app.NewAuthService(c.Store, c.Hasher, c.Clock, c.IDs, c.Config.Auth.SessionTTL)
	})
	return c.authSvc
}

func (c *Core) SubmissionService() *app.SubmissionService {
	return app.NewSubmissionService(c.Store, c.IDs, c.IDs, c.Clock, 5)
}

func (c *Core) ConversationService() *app.ConversationService {
	return app.NewConversationService(c.Store, c.IDs, c.Clock, c.Config.Host.Email, 5)
}

func (c *Core) AttachmentService(ctx context.Context) (*app.AttachmentService, error) {
	objectStorage, err := c.Storage(ctx)
	if err != nil {
		return nil, err
	}
	return app.NewAttachmentService(c.Store, objectStorage, c.IDs, c.Clock,
		c.Config.Limits.MaxUploadBytes, c.Config.Limits.AllowedContentTypes, c.Config.Storage.PresignTTL), nil
}

func (c *Core) Worker() (*app.Worker, error) {
	m, err := c.Mailer()
	if err != nil {
		return nil, err
	}
	return app.NewWorker(c.Store, m, c.IDs, c.Clock, c.Logger, c.Links,
		c.Config.Host.Email, c.Config.Host.Name, c.Config.Worker.PollInterval, c.Config.Worker.Concurrency), nil
}

// API assembles the HTTP transport for the requested service contexts. Only the
// dependencies a context needs are built (for example, no storage client is
// created for the identity or conversation service).
func (c *Core) API(ctx context.Context, enable transporthttp.Enable) (*transporthttp.API, error) {
	opts := transporthttp.Options{
		Config: c.Config,
		Logger: c.Logger,
		Pinger: c.Store.Ping,
		Auth:   c.AuthService(),
		Enable: enable,
	}
	if enable.Submission {
		opts.Submissions = c.SubmissionService()
	}
	// The visitor thread view served by the submission context reads
	// conversations, so build that dependency even when the host-facing
	// conversation routes are not enabled in this process.
	if enable.Submission || enable.Conversation {
		opts.Conversations = c.ConversationService()
	}
	if enable.Attachment {
		attachments, err := c.AttachmentService(ctx)
		if err != nil {
			return nil, err
		}
		opts.Attachments = attachments
	}
	return transporthttp.New(opts), nil
}

// Close releases infrastructure resources.
func (c *Core) Close() {
	if c.Store != nil {
		c.Store.Close()
	}
}
