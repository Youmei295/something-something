package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/youmei295/something-something/src/backend/internal/domain"
)

// handleVisitorThread serves the tokenized, read-only visitor view. The token is
// the capability: no account or session is required.
func (a *API) handleVisitorThread(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	if token == "" {
		writeError(w, domain.ErrNotFound)
		return
	}
	detail, err := a.conversations.GetByVisitorToken(r.Context(), token)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toDetailDTO(detail))
}
