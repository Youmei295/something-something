// Package http implements the HTTP transport (driving adapter). It translates
// requests into application use cases and errors into HTTP responses, and has
// no business logic of its own.
package http

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/youmei295/something-something/src/backend/internal/app"
	"github.com/youmei295/something-something/src/backend/internal/domain"
)

// ErrorResponse is the single error shape returned by every endpoint.
type ErrorResponse struct {
	Error  string            `json:"error"`
	Fields map[string]string `json:"fields,omitempty"`
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if body == nil {
		return
	}
	if err := json.NewEncoder(w).Encode(body); err != nil {
		slog.Default().Error("write response", slog.String("error", err.Error()))
	}
}

func writeError(w http.ResponseWriter, err error) {
	status, message := statusFor(err)

	var ve *app.ValidationError
	if errors.As(err, &ve) {
		writeJSON(w, status, ErrorResponse{Error: message, Fields: ve.Fields})
		return
	}
	writeJSON(w, status, ErrorResponse{Error: message})
}

// statusFor maps domain and validation errors to HTTP status codes. Keeping this
// mapping in one place means endpoints never leak transport-agnostic errors.
func statusFor(err error) (int, string) {
	switch {
	case errors.Is(err, domain.ErrValidation):
		return http.StatusBadRequest, "validation failed"
	case errors.Is(err, domain.ErrUnauthorized):
		return http.StatusUnauthorized, "unauthorized"
	case errors.Is(err, domain.ErrForbidden):
		return http.StatusForbidden, "forbidden"
	case errors.Is(err, domain.ErrNotFound):
		return http.StatusNotFound, "not found"
	case errors.Is(err, domain.ErrConflict):
		return http.StatusConflict, "conflict"
	case errors.Is(err, domain.ErrPayloadTooBig):
		return http.StatusRequestEntityTooLarge, "payload too large"
	case errors.Is(err, domain.ErrUnsupported):
		return http.StatusUnsupportedMediaType, "unsupported media type"
	default:
		return http.StatusInternalServerError, "internal server error"
	}
}

// decodeJSON reads a JSON body with a size cap and rejects unknown fields.
func decodeJSON(w http.ResponseWriter, r *http.Request, dst any, maxBytes int64) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return &app.ValidationError{Fields: map[string]string{"body": "invalid JSON: " + err.Error()}}
	}
	return nil
}
