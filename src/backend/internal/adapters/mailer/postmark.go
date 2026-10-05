package mailer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/youmei295/something-something/src/backend/internal/config"
	"github.com/youmei295/something-something/src/backend/internal/ports"
)

const postmarkEndpoint = "https://api.postmarkapp.com/email"

// Postmark delivers mail through the Postmark transactional API.
type Postmark struct {
	token  string
	from   string
	client *http.Client
}

func NewPostmark(cfg config.MailerConfig) *Postmark {
	from := cfg.FromAddress
	if cfg.FromName != "" {
		from = fmt.Sprintf("%s <%s>", cfg.FromName, cfg.FromAddress)
	}
	return &Postmark{
		token:  cfg.PostmarkToken,
		from:   from,
		client: &http.Client{Timeout: 15 * time.Second},
	}
}

type postmarkRequest struct {
	From     string `json:"From"`
	To       string `json:"To"`
	Subject  string `json:"Subject"`
	TextBody string `json:"TextBody,omitempty"`
	HTMLBody string `json:"HtmlBody,omitempty"`
	ReplyTo  string `json:"ReplyTo,omitempty"`
}

func (p *Postmark) Send(ctx context.Context, e ports.Email) error {
	body := postmarkRequest{
		From:     p.from,
		To:       e.To,
		Subject:  e.Subject,
		TextBody: e.TextBody,
		HTMLBody: e.HTMLBody,
		ReplyTo:  e.ReplyTo,
	}
	buf, err := json.Marshal(body)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, postmarkEndpoint, bytes.NewReader(buf))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Postmark-Server-Token", p.token)

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("postmark: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("postmark: unexpected status %d", resp.StatusCode)
	}
	return nil
}

var _ ports.Mailer = (*Postmark)(nil)
