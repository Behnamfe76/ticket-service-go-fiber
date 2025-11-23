package domain

import "time"

// StaffRole enumerates internal operator roles.
type StaffRole string

const (
	StaffRoleAgent    StaffRole = "AGENT"
	StaffRoleTeamLead StaffRole = "TEAM_LEAD"
	StaffRoleAdmin    StaffRole = "ADMIN"
)

// StaffMember models a support agent or administrator.
type StaffMember struct {
	ID           string
	Name         string
	Email        string
	PasswordHash string
	Role         StaffRole
	DepartmentID *string
	TeamID       *string
	Active       bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
