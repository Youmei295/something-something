// Package link builds absolute URLs for tokenized visitor views.
package link

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/youmei295/something-something/src/backend/internal/ports"
)

// Builder implements ports.VisitorLinkBuilder.
type Builder struct {
	baseURL string
}

func NewBuilder(baseURL string) *Builder {
	return &Builder{baseURL: strings.TrimRight(baseURL, "/")}
}

// ThreadURL returns the public URL where a visitor can read the thread. The
// frontend is a static multi-page site, so the token is passed as a query
// parameter rather than a path segment (no server rewrites required).
func (b *Builder) ThreadURL(token string) string {
	return fmt.Sprintf("%s/visitor.html?token=%s", b.baseURL, url.QueryEscape(token))
}

var _ ports.VisitorLinkBuilder = (*Builder)(nil)
