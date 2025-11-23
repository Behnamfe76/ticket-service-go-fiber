package events

import (
	"time"

	"github.com/spec-kit/ticket-service/internal/domain"
)

// EventType enumerates supported event identifiers.
type EventType string

const (
	EventTicketCreated         EventType = "ticket_created"
	EventTicketStatusChanged   EventType = "ticket_status_changed"
	EventTicketPriorityChanged EventType = "ticket_priority_changed"
	EventTicketAssigned        EventType = "ticket_assigned"
	EventTicketMessageAdded    EventType = "ticket_message_added"
)

// Actor encapsulates actor metadata for an event.
type Actor struct {
	Type    domain.SubjectType `json:"type"`
	UserID  *string            `json:"user_id,omitempty"`
	StaffID *string            `json:"staff_id,omitempty"`
}

// Event represents a domain event emitted by services.
type Event struct {
	ID        string      `json:"id"`
	Type      EventType   `json:"type"`
	TicketID  string      `json:"ticket_id"`
	Actor     Actor       `json:"actor"`
	Timestamp time.Time   `json:"timestamp"`
	Payload   interface{} `json:"payload"`
}

// TicketCreatedPayload payload.
type TicketCreatedPayload struct {
	DepartmentID string                `json:"department_id"`
	TeamID       *string               `json:"team_id,omitempty"`
	Priority     domain.TicketPriority `json:"priority"`
	Title        string                `json:"title"`
}

// TicketStatusChangedPayload payload.
type TicketStatusChangedPayload struct {
	OldStatus domain.TicketStatus `json:"old_status"`
	NewStatus domain.TicketStatus `json:"new_status"`
	Comment   string              `json:"comment,omitempty"`
}

// TicketPriorityChangedPayload payload.
type TicketPriorityChangedPayload struct {
	OldPriority domain.TicketPriority `json:"old_priority"`
	NewPriority domain.TicketPriority `json:"new_priority"`
}

// TicketAssignedPayload payload.
type TicketAssignedPayload struct {
	AssigneeStaffID *string `json:"assignee_staff_id,omitempty"`
	TeamID          *string `json:"team_id,omitempty"`
}

// TicketMessageAddedPayload payload.
type TicketMessageAddedPayload struct {
	MessageID   string                   `json:"message_id"`
	MessageType domain.TicketMessageType `json:"message_type"`
	AuthorType  domain.MessageAuthorType `json:"author_type"`
	AuthorID    *string                  `json:"author_id,omitempty"`
	BodyPreview string                   `json:"body_preview"`
}
