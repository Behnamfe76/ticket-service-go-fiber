package domain

import "time"

// SubjectType differentiates users vs staff tokens.
type SubjectType string

const (
	SubjectTypeUser  SubjectType = "USER"
	SubjectTypeStaff SubjectType = "STAFF"
)

// Token represents issued authentication tokens (JWT or opaque) metadata.
type Token struct {
	ID        string
	SubjectID string
	Subject   SubjectType
	Role      *StaffRole
	ExpiresAt time.Time
	IssuedAt  time.Time
}
