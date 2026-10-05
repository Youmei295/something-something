package http

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/youmei295/something-something/src/backend/internal/domain"
	"github.com/youmei295/something-something/src/backend/internal/ports"
)

func (a *API) handleListConversations(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))
	status := ports.ConversationStatusFilter(q.Get("status"))
	if status == "" {
		status = ports.FilterAll
	}

	items, total, err := a.conversations.List(r.Context(), ports.ListConversationsFilter{
		Status: status,
		Search: q.Get("q"),
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		writeError(w, err)
		return
	}

	dtos := make([]conversationDTO, 0, len(items))
	for _, c := range items {
		dtos = append(dtos, toConversationDTO(c))
	}
	if limit <= 0 || limit > 100 {
		limit = 25
	}
	writeJSON(w, http.StatusOK, listConversationsDTO{Items: dtos, Total: total, Limit: limit, Offset: offset})
}

func (a *API) handleGetConversation(w http.ResponseWriter, r *http.Request) {
	id, err := urlUUID(r, "id")
	if err != nil {
		writeError(w, err)
		return
	}
	detail, err := a.conversations.Get(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toDetailDTO(detail))
}

type replyRequest struct {
	Body          string      `json:"body"`
	AttachmentIDs []uuid.UUID `json:"attachment_ids"`
}

func (a *API) handleReply(w http.ResponseWriter, r *http.Request) {
	id, err := urlUUID(r, "id")
	if err != nil {
		writeError(w, err)
		return
	}
	var req replyRequest
	if err := decodeJSON(w, r, &req, 256<<10); err != nil {
		writeError(w, err)
		return
	}
	message, err := a.conversations.Reply(r.Context(), id, req.Body, req.AttachmentIDs)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, toMessageDTO(*message))
}

type updateConversationRequest struct {
	Read     *bool `json:"read"`
	Archived *bool `json:"archived"`
}

func (a *API) handleUpdateConversation(w http.ResponseWriter, r *http.Request) {
	id, err := urlUUID(r, "id")
	if err != nil {
		writeError(w, err)
		return
	}
	var req updateConversationRequest
	if err := decodeJSON(w, r, &req, 4<<10); err != nil {
		writeError(w, err)
		return
	}
	if req.Read != nil {
		if err := a.conversations.SetRead(r.Context(), id, *req.Read); err != nil {
			writeError(w, err)
			return
		}
	}
	if req.Archived != nil {
		if err := a.conversations.SetArchived(r.Context(), id, *req.Archived); err != nil {
			writeError(w, err)
			return
		}
	}
	detail, err := a.conversations.Get(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toDetailDTO(detail))
}

func (a *API) handleDeleteConversation(w http.ResponseWriter, r *http.Request) {
	id, err := urlUUID(r, "id")
	if err != nil {
		writeError(w, err)
		return
	}
	if err := a.conversations.Delete(r.Context(), id); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// urlUUID parses a UUID path parameter, returning ErrNotFound on bad input so
// invalid IDs and missing resources are indistinguishable to clients.
func urlUUID(r *http.Request, key string) (uuid.UUID, error) {
	id, err := uuid.Parse(chi.URLParam(r, key))
	if err != nil {
		return uuid.Nil, domain.ErrNotFound
	}
	return id, nil
}
