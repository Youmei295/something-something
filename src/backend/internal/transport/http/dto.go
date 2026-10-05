package http

import (
	"time"

	"github.com/google/uuid"

	"github.com/youmei295/something-something/src/backend/internal/app"
	"github.com/youmei295/something-something/src/backend/internal/domain"
)

// DTOs define the stable JSON contract exposed to clients, independent from the
// internal domain model so refactors do not break the API.

type userDTO struct {
	ID    uuid.UUID `json:"id"`
	Email string    `json:"email"`
	Role  string    `json:"role"`
}

type conversationDTO struct {
	ID             uuid.UUID `json:"id"`
	Subject        string    `json:"subject"`
	Status         string    `json:"status"`
	VisitorName    string    `json:"visitor_name"`
	VisitorEmail   string    `json:"visitor_email"`
	Unread         bool      `json:"unread"`
	CreatedAt      time.Time `json:"created_at"`
	LastActivityAt time.Time `json:"last_activity_at"`
}

type messageDTO struct {
	ID        uuid.UUID `json:"id"`
	Direction string    `json:"direction"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
}

type attachmentDTO struct {
	ID          uuid.UUID `json:"id"`
	Filename    string    `json:"filename"`
	ContentType string    `json:"content_type"`
	Size        int64     `json:"size"`
	ScanStatus  string    `json:"scan_status"`
}

type conversationDetailDTO struct {
	Conversation conversationDTO `json:"conversation"`
	Messages     []messageDTO    `json:"messages"`
	Attachments  []attachmentDTO `json:"attachments"`
}

type listConversationsDTO struct {
	Items  []conversationDTO `json:"items"`
	Total  int               `json:"total"`
	Limit  int               `json:"limit"`
	Offset int               `json:"offset"`
}

func toUserDTO(u *domain.User) userDTO {
	return userDTO{ID: u.ID, Email: u.Email, Role: u.Role}
}

func toConversationDTO(c domain.Conversation) conversationDTO {
	return conversationDTO{
		ID: c.ID, Subject: c.Subject, Status: string(c.Status),
		VisitorName: c.VisitorName, VisitorEmail: c.VisitorEmail, Unread: c.Unread,
		CreatedAt: c.CreatedAt, LastActivityAt: c.LastActivityAt,
	}
}

func toMessageDTO(m domain.Message) messageDTO {
	return messageDTO{ID: m.ID, Direction: string(m.Direction), Body: m.Body, CreatedAt: m.CreatedAt}
}

func toAttachmentDTO(a domain.Attachment) attachmentDTO {
	return attachmentDTO{
		ID: a.ID, Filename: a.Filename, ContentType: a.ContentType,
		Size: a.Size, ScanStatus: string(a.ScanStatus),
	}
}

func toDetailDTO(d *app.ConversationDetail) conversationDetailDTO {
	messages := make([]messageDTO, 0, len(d.Messages))
	for _, m := range d.Messages {
		messages = append(messages, toMessageDTO(m))
	}
	attachments := make([]attachmentDTO, 0, len(d.Attachments))
	for _, a := range d.Attachments {
		attachments = append(attachments, toAttachmentDTO(a))
	}
	return conversationDetailDTO{
		Conversation: toConversationDTO(d.Conversation),
		Messages:     messages,
		Attachments:  attachments,
	}
}
