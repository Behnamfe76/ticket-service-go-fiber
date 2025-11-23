package dto

import "github.com/spec-kit/ticket-service/internal/domain"

// StaffLoginRequest payload.
type StaffLoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// PasswordResetRequest payload for initiating reset.
type PasswordResetRequest struct {
	Email string `json:"email"`
}

// PasswordResetConfirmRequest payload for confirming reset.
type PasswordResetConfirmRequest struct {
	Token       string `json:"token"`
	NewPassword string `json:"new_password"`
}

// PasswordChangeRequest payload for authenticated password changes.
type PasswordChangeRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

// DepartmentRequest for create/update.
type DepartmentRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	IsActive    *bool  `json:"is_active,omitempty"`
}

// DepartmentResponse representation.
type DepartmentResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	IsActive    bool   `json:"is_active"`
}

// TeamRequest for create/update operations.
type TeamRequest struct {
	DepartmentID string `json:"department_id"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	IsActive     *bool  `json:"is_active,omitempty"`
}

// TeamResponse payload.
type TeamResponse struct {
	ID           string `json:"id"`
	DepartmentID string `json:"department_id"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	IsActive     bool   `json:"is_active"`
}

// StaffCreateRequest payload for new staff.
type StaffCreateRequest struct {
	Name     string           `json:"name"`
	Email    string           `json:"email"`
	Password string           `json:"password"`
	Role     domain.StaffRole `json:"role"`
	TeamID   *string          `json:"team_id"`
}

// StaffUpdateRequest payload for updates.
type StaffUpdateRequest struct {
	Name   string           `json:"name"`
	Email  string           `json:"email"`
	Role   domain.StaffRole `json:"role"`
	TeamID *string          `json:"team_id"`
	Active bool             `json:"active"`
}

// StaffResponse representation.
type StaffResponse struct {
	ID           string           `json:"id"`
	Name         string           `json:"name"`
	Email        string           `json:"email"`
	Role         domain.StaffRole `json:"role"`
	DepartmentID *string          `json:"department_id"`
	TeamID       *string          `json:"team_id"`
	Active       bool             `json:"active"`
}

// StaffListQuery query params.
type StaffListQuery struct {
	Role         *domain.StaffRole
	TeamID       *string
	DepartmentID *string
	Active       *bool
	Page         int
	PageSize     int
}
