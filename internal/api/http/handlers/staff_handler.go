package handlers

import (
	"net/http"

	"github.com/gofiber/fiber/v2"

	"github.com/spec-kit/ticket-service/internal/api/dto"
	"github.com/spec-kit/ticket-service/internal/auth"
	"github.com/spec-kit/ticket-service/internal/domain"
	"github.com/spec-kit/ticket-service/internal/service"
)

// StaffHandler exposes staff/auth endpoints.
type StaffHandler struct {
	authService *service.AuthService
}

// NewStaffHandler constructs handler.
func NewStaffHandler(authService *service.AuthService) *StaffHandler {
	return &StaffHandler{authService: authService}
}

// Login handles POST /auth/staff/login.
func (h *StaffHandler) Login(c *fiber.Ctx) error {
	var req dto.StaffLoginRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(http.StatusBadRequest, "invalid payload")
	}
	if req.Email == "" || req.Password == "" {
		return fiber.NewError(http.StatusBadRequest, "email and password required")
	}

	staff, token, exp, err := h.authService.LoginStaff(c.Context(), req.Email, req.Password)
	if err != nil {
		return fiber.NewError(http.StatusUnauthorized, err.Error())
	}

	return c.JSON(fiber.Map{
		"data": fiber.Map{
			"staff": fiber.Map{
				"id":    staff.ID,
				"name":  staff.Name,
				"email": staff.Email,
				"role":  staff.Role,
			},
			"auth": dto.AuthResponse{Token: token, ExpiresAt: exp},
		},
	})
}

// RequestPasswordReset handles POST /auth/password/reset/request.
func (h *StaffHandler) RequestPasswordReset(c *fiber.Ctx) error {
	var req dto.PasswordResetRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(http.StatusBadRequest, "invalid payload")
	}
	if req.Email == "" {
		return fiber.NewError(http.StatusBadRequest, "email required")
	}

	token, err := h.authService.RequestPasswordReset(c.Context(), req.Email)
	if err != nil {
		return fiber.NewError(http.StatusBadRequest, err.Error())
	}
	return c.Status(http.StatusAccepted).JSON(fiber.Map{
		"data": fiber.Map{
			"reset_token": token.Token,
			"expires_at":  token.ExpiresAt,
		},
	})
}

// ConfirmPasswordReset handles POST /auth/password/reset/confirm.
func (h *StaffHandler) ConfirmPasswordReset(c *fiber.Ctx) error {
	var req dto.PasswordResetConfirmRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(http.StatusBadRequest, "invalid payload")
	}
	if req.Token == "" || req.NewPassword == "" {
		return fiber.NewError(http.StatusBadRequest, "token and new password required")
	}

	if err := h.authService.ConfirmPasswordReset(c.Context(), req.Token, req.NewPassword); err != nil {
		return fiber.NewError(http.StatusBadRequest, err.Error())
	}
	return c.JSON(fiber.Map{"data": fiber.Map{"status": "password_reset"}})
}

// ChangePassword handles POST /auth/password/change.
func (h *StaffHandler) ChangePassword(c *fiber.Ctx) error {
	principal, ok := auth.PrincipalFromContext(c)
	if !ok {
		return fiber.NewError(http.StatusUnauthorized, "authentication required")
	}

	var req dto.PasswordChangeRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(http.StatusBadRequest, "invalid payload")
	}
	if req.CurrentPassword == "" || req.NewPassword == "" {
		return fiber.NewError(http.StatusBadRequest, "current and new password required")
	}

	subject := service.AuthSubject{Type: principal.SubjectType}
	switch principal.SubjectType {
	case domain.SubjectTypeUser:
		subject.ID = principal.User.ID
	case domain.SubjectTypeStaff:
		subject.ID = principal.Staff.ID
	default:
		return fiber.NewError(http.StatusUnauthorized, "unknown subject")
	}

	if err := h.authService.ChangePassword(c.Context(), subject, req.CurrentPassword, req.NewPassword); err != nil {
		return fiber.NewError(http.StatusBadRequest, err.Error())
	}
	return c.JSON(fiber.Map{"data": fiber.Map{"status": "password_changed"}})
}
