package mailer

import (
	"context"
	"fmt"
	"net"
	"net/smtp"
	"strings"

	"github.com/youmei295/something-something/src/backend/internal/config"
	"github.com/youmei295/something-something/src/backend/internal/ports"
)

// SMTP delivers mail through a classic SMTP relay.
type SMTP struct {
	cfg config.MailerConfig
}

func NewSMTP(cfg config.MailerConfig) *SMTP { return &SMTP{cfg: cfg} }

func (s *SMTP) Send(_ context.Context, e ports.Email) error {
	addr := net.JoinHostPort(s.cfg.SMTPHost, fmt.Sprintf("%d", s.cfg.SMTPPort))
	from := s.cfg.FromAddress

	var auth smtp.Auth
	if s.cfg.SMTPUsername != "" {
		auth = smtp.PlainAuth("", s.cfg.SMTPUsername, s.cfg.SMTPPassword, s.cfg.SMTPHost)
	}

	raw := s.buildMessage(e)
	return smtp.SendMail(addr, auth, from, []string{e.To}, raw)
}

func (s *SMTP) buildMessage(e ports.Email) []byte {
	var b strings.Builder
	fromName := s.cfg.FromName
	if fromName != "" {
		fmt.Fprintf(&b, "From: %s <%s>\r\n", fromName, s.cfg.FromAddress)
	} else {
		fmt.Fprintf(&b, "From: %s\r\n", s.cfg.FromAddress)
	}
	fmt.Fprintf(&b, "To: %s\r\n", e.To)
	fmt.Fprintf(&b, "Subject: %s\r\n", e.Subject)
	b.WriteString("MIME-Version: 1.0\r\n")
	if e.ReplyTo != "" {
		fmt.Fprintf(&b, "Reply-To: %s\r\n", e.ReplyTo)
	}
	for k, v := range e.Headers {
		fmt.Fprintf(&b, "%s: %s\r\n", k, v)
	}
	if e.HTMLBody != "" {
		b.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
		b.WriteString("\r\n")
		b.WriteString(e.HTMLBody)
	} else {
		b.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
		b.WriteString("\r\n")
		b.WriteString(e.TextBody)
	}
	return []byte(b.String())
}

// Note: net/smtp negotiates STARTTLS automatically when the server advertises it.
var _ ports.Mailer = (*SMTP)(nil)
