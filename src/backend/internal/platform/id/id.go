// Package id provides concrete ID/token generation adapters.
package id

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/youmei295/something-something/src/backend/internal/ports"
)

// Generator implements ports.IDGenerator.
type Generator struct{}

func NewGenerator() *Generator { return &Generator{} }

func (Generator) New() uuid.UUID { return uuid.New() }

// Generator also implements ports.TokenGenerator. NewToken returns a URL-safe
// random token used for visitor thread links.
func (Generator) NewToken(n int) (string, error) { return NewOpaqueToken(n) }

// NewOpaqueToken returns a URL-safe random token used for visitor thread links.
func NewOpaqueToken(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

// NewEmailMessageID constructs a plausible RFC 5322 Message-ID for outbound mail
// so replies can be threaded by mail clients.
func NewEmailMessageID(domain string) string {
	tok, err := NewOpaqueToken(12)
	if err != nil {
		tok = uuid.NewString()
	}
	return fmt.Sprintf("<%d.%s@%s>", time.Now().UnixNano(), tok, domain)
}

var (
	_ ports.IDGenerator    = Generator{}
	_ ports.TokenGenerator = Generator{}
)
