package http

import (
	"github.com/gofiber/fiber/v2"

	"github.com/spec-kit/ticket-service/internal/api/http/handlers"
)

// RegisterRoutes wires HTTP routes.
func RegisterRoutes(app *fiber.App, healthHandler *handlers.HealthHandler) {
	app.Get("/health/live", healthHandler.Live)
	app.Get("/health/ready", healthHandler.Ready)
}
