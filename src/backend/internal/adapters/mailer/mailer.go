// Package mailer provides ports.Mailer implementations: console (development),
// SMTP, and Postmark (transactional provider). The application only sees the
// interface, so switching providers is a wiring decision.
package mailer

import (
	"log/slog"

	"github.com/youmei295/something-something/src/backend/internal/config"
	"github.com/youmei295/something-something/src/backend/internal/ports"
)

// New selects a mailer from configuration.
func New(cfg config.MailerConfig, log *slog.Logger) (ports.Mailer, error) {
	switch cfg.Driver {
	case "smtp":
		return NewSMTP(cfg), nil
	case "postmark":
		return NewPostmark(cfg), nil
	default:
		return NewConsole(log), nil
	}
}
