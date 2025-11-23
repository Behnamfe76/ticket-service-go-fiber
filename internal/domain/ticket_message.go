package domain

import "time"

// MessageAuthorType indicates who authored a message.
type MessageAuthorType string

const (
	AuthorTypeUser   MessageAuthorType = "USER"
	AuthorTypeStaff  MessageAuthorType = "STAFF"
	AuthorTypeSystem MessageAuthorType = "SYSTEM"
)

// TicketMessageType differentiates between replies and notes.
type TicketMessageType string

const (
	MessageTypePublicReply  TicketMessageType = "PUBLIC_REPLY"
	MessageTypeInternalNote TicketMessageType = "INTERNAL_NOTE"
	MessageTypeSystemEvent  TicketMessageType = "SYSTEM_EVENT"
)

// TicketMessage captures communications in a ticket thread.
type TicketMessage struct {
	ID          string
	TicketID    string
	AuthorType  MessageAuthorType
	AuthorID    *string
	MessageType TicketMessageType
	Body        string
	Attachments []AttachmentReference
	CreatedAt   time.Time
}

// AttachmentReference stores metadata for ticket message attachments.
type AttachmentReference struct {
	ID              string
	TicketMessageID string
	StorageKey      string
	FileName        string
	MimeType        string
	SizeBytes       int64
	CreatedAt       time.Time
}
