package domain

import "time"

// TicketStatus enumerates lifecycle states for tickets.
type TicketStatus string

const (
	TicketStatusOpen        TicketStatus = "OPEN"
	TicketStatusInProgress  TicketStatus = "IN_PROGRESS"
	TicketStatusPendingUser TicketStatus = "PENDING_USER"
	TicketStatusResolved    TicketStatus = "RESOLVED"
	TicketStatusClosed      TicketStatus = "CLOSED"
	TicketStatusCancelled   TicketStatus = "CANCELLED"
)

// TicketPriority enumerates SLA urgency.
type TicketPriority string

const (
	TicketPriorityLow    TicketPriority = "LOW"
	TicketPriorityMedium TicketPriority = "MEDIUM"
	TicketPriorityHigh   TicketPriority = "HIGH"
	TicketPriorityUrgent TicketPriority = "URGENT"
)

// Ticket is the aggregate for support requests.
type Ticket struct {
	ID           string
	ExternalKey  string
	RequesterID  string
	DepartmentID string
	TeamID       *string
	AssigneeID   *string
	Title        string
	Description  string
	Status       TicketStatus
	Priority     TicketPriority
	Tags         []string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	ClosedAt     *time.Time
}
