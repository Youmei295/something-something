// Package link builds absolute URLs for tokenized visitor views.
package link

import (
	"fmt"
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

// ThreadURL returns the public URL where a visitor can read the thread.
func (b *Builder) ThreadURL(token string) string {
	return fmt.Sprintf("%s/t/%s", b.baseURL, token)
}

var _ ports.VisitorLinkBuilder = (*Builder)(nil)
