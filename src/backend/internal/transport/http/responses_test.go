package http

import (
	"errors"
	"net/http"
	"testing"

	"github.com/youmei295/something-something/src/backend/internal/app"
	"github.com/youmei295/something-something/src/backend/internal/domain"
)

func TestStatusFor(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want int
	}{
		{"validation", &app.ValidationError{Fields: map[string]string{"x": "y"}}, http.StatusBadRequest},
		{"unauthorized", domain.ErrUnauthorized, http.StatusUnauthorized},
		{"forbidden", domain.ErrForbidden, http.StatusForbidden},
		{"not found", domain.ErrNotFound, http.StatusNotFound},
		{"conflict", domain.ErrConflict, http.StatusConflict},
		{"too large", domain.ErrPayloadTooBig, http.StatusRequestEntityTooLarge},
		{"unsupported", domain.ErrUnsupported, http.StatusUnsupportedMediaType},
		{"unknown", errors.New("boom"), http.StatusInternalServerError},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, _ := statusFor(tc.err)
			if got != tc.want {
				t.Fatalf("statusFor = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestValidationError_Is(t *testing.T) {
	err := &app.ValidationError{Fields: map[string]string{"email": "required"}}
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatal("ValidationError should match domain.ErrValidation")
	}
	if errors.Is(err, domain.ErrNotFound) {
		t.Fatal("ValidationError should not match unrelated errors")
	}
}
