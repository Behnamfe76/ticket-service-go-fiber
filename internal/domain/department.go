package domain

import "time"

// Department represents a high-level organizational unit.
type Department struct {
	ID          string
	Name        string
	Description string
	IsActive    bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
