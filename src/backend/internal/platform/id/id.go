// Package id provides concrete ID/token generation adapters.
package id

import (
	"crypto/rand"
	"encoding/base64"

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

var (
	_ ports.IDGenerator    = Generator{}
	_ ports.TokenGenerator = Generator{}
)
