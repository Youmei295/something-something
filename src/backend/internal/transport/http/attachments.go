package http

import (
	"net/http"

	"github.com/youmei295/something-something/src/backend/internal/app"
)

type requestUploadRequest struct {
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	Size        int64  `json:"size"`
}

func (a *API) handleRequestUpload(w http.ResponseWriter, r *http.Request) {
	var req requestUploadRequest
	if err := decodeJSON(w, r, &req, 16<<10); err != nil {
		writeError(w, err)
		return
	}
	ticket, err := a.attachments.RequestUpload(r.Context(), app.RequestUploadInput{
		Filename:    req.Filename,
		ContentType: req.ContentType,
		Size:        req.Size,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, ticket)
}

func (a *API) handleCompleteUpload(w http.ResponseWriter, r *http.Request) {
	id, err := urlUUID(r, "id")
	if err != nil {
		writeError(w, err)
		return
	}
	attachment, err := a.attachments.CompleteUpload(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toAttachmentDTO(*attachment))
}

func (a *API) handleDownloadAttachment(w http.ResponseWriter, r *http.Request) {
	id, err := urlUUID(r, "id")
	if err != nil {
		writeError(w, err)
		return
	}
	url, err := a.attachments.DownloadURL(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"url": url})
}
