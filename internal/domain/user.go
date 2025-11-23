package domain

import "time"

// UserStatus represents lifecycle states for an end-user.
type UserStatus string

const (
	UserStatusActive    UserStatus = "ACTIVE"
	UserStatusSuspended UserStatus = "SUSPENDED"
)

// User is the domain model for end-users who submit tickets.
type User struct {
	ID           string
	Name         string
	Email        string
	PasswordHash string
	Status       UserStatus
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
