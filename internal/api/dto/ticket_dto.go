package dto

import (
	"time"

	"github.com/spec-kit/ticket-service/internal/domain"
)

// CreateTicketRequest payload.
type CreateTicketRequest struct {
	DepartmentID string                `json:"department_id"`
	TeamID       *string               `json:"team_id"`
	Title        string                `json:"title"`
	Description  string                `json:"description"`
	Priority     domain.TicketPriority `json:"priority"`
	Tags         []string              `json:"tags"`
}

// TicketListQuery captures query filters for user endpoints.
type TicketListQuery struct {
	Statuses    []domain.TicketStatus
	Priorities  []domain.TicketPriority
	CreatedFrom *time.Time
	CreatedTo   *time.Time
	Page        int
	PageSize    int
}

// TicketSummary response.
type TicketSummary struct {
	ID           string                `json:"id"`
	ExternalKey  string                `json:"external_key"`
	DepartmentID string                `json:"department_id"`
	TeamID       *string               `json:"team_id"`
	Title        string                `json:"title"`
	Status       domain.TicketStatus   `json:"status"`
	Priority     domain.TicketPriority `json:"priority"`
	Tags         []string              `json:"tags"`
	CreatedAt    time.Time             `json:"created_at"`
	UpdatedAt    time.Time             `json:"updated_at"`
}

// TicketDetailResponse provides full ticket info.
type TicketDetailResponse struct {
	ID           string                  `json:"id"`
	ExternalKey  string                  `json:"external_key"`
	DepartmentID string                  `json:"department_id"`
	TeamID       *string                 `json:"team_id"`
	Title        string                  `json:"title"`
	Description  string                  `json:"description"`
	Status       domain.TicketStatus     `json:"status"`
	Priority     domain.TicketPriority   `json:"priority"`
	Tags         []string                `json:"tags"`
	CreatedAt    time.Time               `json:"created_at"`
	UpdatedAt    time.Time               `json:"updated_at"`
	ClosedAt     *time.Time              `json:"closed_at"`
	Messages     []TicketMessageResponse `json:"messages"`
	History      []TicketHistoryResponse `json:"history"`
}

// TicketMessageResponse represents thread message.
type TicketMessageResponse struct {
	ID          string                   `json:"id"`
	MessageType domain.TicketMessageType `json:"message_type"`
	AuthorType  domain.MessageAuthorType `json:"author_type"`
	AuthorID    *string                  `json:"author_id"`
	Body        string                   `json:"body"`
	Attachments []AttachmentResponse     `json:"attachments"`
	CreatedAt   time.Time                `json:"created_at"`
}

// AttachmentResponse metadata.
type AttachmentResponse struct {
	ID        string `json:"id"`
	FileName  string `json:"file_name"`
	MimeType  string `json:"mime_type"`
	SizeBytes int64  `json:"size_bytes"`
	URL       string `json:"url,omitempty"`
}

// CreateMessageRequest payload.
type CreateMessageRequest struct {
	Body        string                    `json:"body"`
	MessageType *domain.TicketMessageType `json:"message_type,omitempty"`
	Attachments []AttachmentRequest       `json:"attachments"`
}

// AttachmentRequest describes attachment input.
type AttachmentRequest struct {
	StorageKey string `json:"storage_key"`
	FileName   string `json:"file_name"`
	MimeType   string `json:"mime_type"`
	SizeBytes  int64  `json:"size_bytes"`
}

// UpdateStatusRequest payload.
type UpdateStatusRequest struct {
	NewStatus domain.TicketStatus `json:"new_status"`
	Comment   string              `json:"comment"`
}

// UpdatePriorityRequest payload.
type UpdatePriorityRequest struct {
	NewPriority domain.TicketPriority `json:"new_priority"`
}

// AssignStaffRequest payload.
type AssignStaffRequest struct {
	AssigneeStaffID string `json:"assignee_staff_id"`
}

// AssignTeamRequest payload.
type AssignTeamRequest struct {
	TeamID string `json:"team_id"`
}

// TicketHistoryResponse for API responses.
type TicketHistoryResponse struct {
	ID            string                   `json:"id"`
	ChangeType    domain.TicketChangeType  `json:"change_type"`
	ChangedByType domain.MessageAuthorType `json:"changed_by_type"`
	ChangedByID   *string                  `json:"changed_by_id"`
	OldValue      map[string]any           `json:"old_value"`
	NewValue      map[string]any           `json:"new_value"`
	CreatedAt     time.Time                `json:"created_at"`
}
