package http

import (
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/youmei295/something-something/src/backend/internal/app"
)

type submitMessageRequest struct {
	Name          string      `json:"name"`
	Email         string      `json:"email"`
	Subject       string      `json:"subject"`
	Body          string      `json:"body"`
	AttachmentIDs []uuid.UUID `json:"attachment_ids"`
}

type submitMessageResponse struct {
	ConversationID uuid.UUID `json:"conversation_id"`
	VisitorToken   string    `json:"visitor_token"`
	CreatedAt      time.Time `json:"created_at"`
}

func (a *API) handleSubmitMessage(w http.ResponseWriter, r *http.Request) {
	var req submitMessageRequest
	if err := decodeJSON(w, r, &req, 256<<10); err != nil {
		writeError(w, err)
		return
	}

	result, err := a.submissions.Submit(r.Context(), app.SubmitInput{
		Name:          req.Name,
		Email:         req.Email,
		Subject:       req.Subject,
		Body:          req.Body,
		AttachmentIDs: req.AttachmentIDs,
	})
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, submitMessageResponse{
		ConversationID: result.ConversationID,
		VisitorToken:   result.VisitorToken,
		CreatedAt:      result.CreatedAt,
	})
}
