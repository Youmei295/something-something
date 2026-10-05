package http

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/youmei295/something-something/src/backend/internal/app"
	"github.com/youmei295/something-something/src/backend/internal/config"
)

// Enable selects which bounded context this process serves. The same HTTP
// transport is shared by the all-in-one API and by each standalone service.
type Enable struct {
	Identity     bool
	Submission   bool
	Conversation bool
	Attachment   bool
}

// Options is the dependency bundle for New. A service only sets the fields its
// enabled route groups need; unused services may remain nil.
type Options struct {
	Config *config.Config
	Logger *slog.Logger
	Pinger func(ctx context.Context) error

	Auth          *app.AuthService
	Submissions   *app.SubmissionService
	Conversations *app.ConversationService
	Attachments   *app.AttachmentService

	Enable Enable
}

// API holds the handlers and their dependencies.
type API struct {
	cfg           *config.Config
	log           *slog.Logger
	pinger        func(ctx context.Context) error
	auth          *app.AuthService
	submissions   *app.SubmissionService
	conversations *app.ConversationService
	attachments   *app.AttachmentService
	enable        Enable
}

func New(opts Options) *API {
	return &API{
		cfg:           opts.Config,
		log:           opts.Logger,
		pinger:        opts.Pinger,
		auth:          opts.Auth,
		submissions:   opts.Submissions,
		conversations: opts.Conversations,
		attachments:   opts.Attachments,
		enable:        opts.Enable,
	}
}

// Router builds the HTTP routing tree for the enabled contexts.
func (a *API) Router() http.Handler {
	r := chi.NewRouter()

	r.Use(requestID)
	r.Use(logRequests(a.log))
	r.Use(recoverPanic(a.log))
	r.Use(cors(a.cfg.HTTP.AllowedOrigins))

	r.Get("/healthz", a.handleHealth)
	r.Get("/readyz", a.handleReady)

	r.Route("/api/v1", func(r chi.Router) {
		// ---- public (visitor-facing) endpoints ----
		if a.enable.Identity {
			r.Post("/auth/login", a.handleLogin)
		}
		if a.enable.Submission {
			r.Post("/messages", a.handleSubmitMessage)
			r.Get("/visitor/threads/{token}", a.handleVisitorThread)
		}
		if a.enable.Attachment {
			r.Post("/attachments/upload-url", a.handleRequestUpload)
			r.Post("/attachments/{id}/complete", a.handleCompleteUpload)
		}

		// ---- host-only endpoints ----
		if a.hasProtectedRoutes() {
			r.Group(func(r chi.Router) {
				r.Use(a.requireAuth)

				if a.enable.Identity {
					r.Post("/auth/logout", a.handleLogout)
					r.Get("/auth/me", a.handleMe)
				}
				if a.enable.Conversation {
					r.Get("/conversations", a.handleListConversations)
					r.Get("/conversations/{id}", a.handleGetConversation)
					r.Post("/conversations/{id}/replies", a.handleReply)
					r.Patch("/conversations/{id}", a.handleUpdateConversation)
					r.Delete("/conversations/{id}", a.handleDeleteConversation)
				}
				if a.enable.Attachment {
					r.Get("/attachments/{id}/download", a.handleDownloadAttachment)
				}
			})
		}
	})

	return r
}

func (a *API) hasProtectedRoutes() bool {
	return a.enable.Identity || a.enable.Conversation || a.enable.Attachment
}
