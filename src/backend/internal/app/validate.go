// Package app contains the application use cases. Services orchestrate domain
// objects and ports; they know nothing about HTTP, SQL, or any concrete vendor.
package app

import (
	"fmt"
	"strings"

	"github.com/youmei295/something-something/src/backend/internal/domain"
)

// ValidationError carries per-field messages. It reports as domain.ErrValidation
// so transports can map it to a 400 without importing app internals.
type ValidationError struct {
	Fields map[string]string
}

func (e *ValidationError) Error() string {
	parts := make([]string, 0, len(e.Fields))
	for k, v := range e.Fields {
		parts = append(parts, k+": "+v)
	}
	return "validation failed: " + strings.Join(parts, "; ")
}

func (e *ValidationError) Is(target error) bool { return target == domain.ErrValidation }

// Validate accumulates field errors and returns a *ValidationError when any
// were recorded, making multi-field validation concise.
type Validate struct {
	fields map[string]string
}

func NewValidate() *Validate { return &Validate{fields: map[string]string{}} }

func (v *Validate) Add(field, msg string) {
	if _, ok := v.fields[field]; !ok {
		v.fields[field] = msg
	}
}

func (v *Validate) Required(field, value string) {
	if strings.TrimSpace(value) == "" {
		v.Add(field, "is required")
	}
}

func (v *Validate) MaxLen(field, value string, n int) {
	if len([]rune(value)) > n {
		v.Add(field, fmt.Sprintf("must be at most %d characters", n))
	}
}

func (v *Validate) MaxItems(field string, n, max int) {
	if n > max {
		v.Add(field, fmt.Sprintf("must contain at most %d items", max))
	}
}

func (v *Validate) Email(field, value string) {
	s := strings.TrimSpace(value)
	if s == "" {
		return
	}
	at := strings.IndexByte(s, '@')
	if at <= 0 || at == len(s)-1 || !strings.Contains(s[at+1:], ".") {
		v.Add(field, "must be a valid email address")
	}
}

// Err returns nil or a *ValidationError.
func (v *Validate) Err() error {
	if len(v.fields) == 0 {
		return nil
	}
	return &ValidationError{Fields: v.fields}
}
