package handlers

import "github.com/gofiber/fiber/v2"

// HealthHandler responds to liveness and readiness probes.
type HealthHandler struct{}

// NewHealthHandler returns a new handler instance.
func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

// Live reports service liveness.
func (h *HealthHandler) Live(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"status": "ok"})
}

// Ready reports service readiness. Placeholder until dependency checks wired.
func (h *HealthHandler) Ready(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"status": "ok"})
}
