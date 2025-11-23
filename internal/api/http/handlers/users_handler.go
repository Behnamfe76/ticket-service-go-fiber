package handlers

import (
	"net/http"

	"github.com/gofiber/fiber/v2"

	"github.com/spec-kit/ticket-service/internal/api/dto"
	"github.com/spec-kit/ticket-service/internal/service"
)

// UsersHandler exposes auth endpoints for end-users.
type UsersHandler struct {
	auth *service.AuthService
}

// NewUsersHandler constructs handler.
func NewUsersHandler(authService *service.AuthService) *UsersHandler {
	return &UsersHandler{auth: authService}
}

// Register handles POST /auth/users/register.
func (h *UsersHandler) Register(c *fiber.Ctx) error {
	var req dto.UserRegisterRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(http.StatusBadRequest, "invalid payload")
	}
	if req.Email == "" || req.Password == "" || req.Name == "" {
		return fiber.NewError(http.StatusBadRequest, "name, email, password required")
	}

	user, token, exp, err := h.auth.RegisterUser(c.Context(), req.Name, req.Email, req.Password)
	if err != nil {
		return fiber.NewError(http.StatusBadRequest, err.Error())
	}

	return c.Status(http.StatusCreated).JSON(fiber.Map{
		"data": fiber.Map{
			"user": fiber.Map{
				"id":    user.ID,
				"name":  user.Name,
				"email": user.Email,
			},
			"auth": dto.AuthResponse{Token: token, ExpiresAt: exp},
		},
	})
}

// Login handles POST /auth/users/login.
func (h *UsersHandler) Login(c *fiber.Ctx) error {
	var req dto.UserLoginRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(http.StatusBadRequest, "invalid payload")
	}
	if req.Email == "" || req.Password == "" {
		return fiber.NewError(http.StatusBadRequest, "email and password required")
	}

	user, token, exp, err := h.auth.LoginUser(c.Context(), req.Email, req.Password)
	if err != nil {
		return fiber.NewError(http.StatusUnauthorized, err.Error())
	}

	return c.JSON(fiber.Map{
		"data": fiber.Map{
			"user": fiber.Map{
				"id":    user.ID,
				"name":  user.Name,
				"email": user.Email,
			},
			"auth": dto.AuthResponse{Token: token, ExpiresAt: exp},
		},
	})
}
