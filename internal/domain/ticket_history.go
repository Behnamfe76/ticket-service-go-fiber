package domain

import "time"

// TicketChangeType captures what changed in a history entry.
type TicketChangeType string

const (
	ChangeTypeStatus     TicketChangeType = "STATUS_CHANGE"
	ChangeTypeAssignee   TicketChangeType = "ASSIGNEE_CHANGE"
	ChangeTypePriority   TicketChangeType = "PRIORITY_CHANGE"
	ChangeTypeTeam       TicketChangeType = "TEAM_CHANGE"
	ChangeTypeDepartment TicketChangeType = "DEPARTMENT_CHANGE"
	ChangeTypeTags       TicketChangeType = "TAGS_CHANGE"
)

// TicketHistory is an immutable audit trail entry.
type TicketHistory struct {
	ID            string
	TicketID      string
	ChangedByType MessageAuthorType
	ChangedByID   *string
	ChangeType    TicketChangeType
	OldValue      map[string]any
	NewValue      map[string]any
	CreatedAt     time.Time
}
