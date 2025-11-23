package domain

import "time"

// Team represents a sub-group under a department.
type Team struct {
	ID           string
	DepartmentID string
	Name         string
	Description  string
	IsActive     bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
