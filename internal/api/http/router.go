package http

import (
	"github.com/gofiber/fiber/v2"

	"github.com/spec-kit/ticket-service/internal/api/http/handlers"
	"github.com/spec-kit/ticket-service/internal/auth"
)

// RouteConfig bundles dependencies for route registration.
type RouteConfig struct {
	Health         *handlers.HealthHandler
	Users          *handlers.UsersHandler
	Staff          *handlers.StaffHandler
	AuthMiddleware *auth.AuthMiddleware
}

// RegisterRoutes wires HTTP routes.
func RegisterRoutes(app *fiber.App, cfg RouteConfig) {
	app.Get("/health/live", cfg.Health.Live)
	app.Get("/health/ready", cfg.Health.Ready)

	authGroup := app.Group("/auth")
	authGroup.Post("/users/register", cfg.Users.Register)
	authGroup.Post("/users/login", cfg.Users.Login)

	authGroup.Post("/staff/login", cfg.Staff.Login)
	authGroup.Post("/password/reset/request", cfg.Staff.RequestPasswordReset)
	authGroup.Post("/password/reset/confirm", cfg.Staff.ConfirmPasswordReset)

	protected := authGroup.Group("", cfg.AuthMiddleware.Handle, auth.RequireAnyRole())
	protected.Post("/password/change", cfg.Staff.ChangePassword)
}
