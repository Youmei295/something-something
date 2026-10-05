package mailer

import (
	"context"
	"log/slog"

	"github.com/youmei295/something-something/src/backend/internal/ports"
)

// Console logs emails instead of sending them. Useful in development and tests.
type Console struct {
	log *slog.Logger
}

func NewConsole(log *slog.Logger) *Console { return &Console{log: log} }

func (c *Console) Send(_ context.Context, e ports.Email) error {
	c.log.Info("email (console)",
		slog.String("to", e.To),
		slog.String("subject", e.Subject),
		slog.String("body", e.TextBody),
	)
	return nil
}

var _ ports.Mailer = (*Console)(nil)
