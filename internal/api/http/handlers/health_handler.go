package handlers

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/spec-kit/ticket-service/internal/persistence"
)

// HealthHandler responds to liveness and readiness probes.
type HealthHandler struct {
	serviceName string
	version     string
	postgres    *persistence.Postgres
	redis       *persistence.Redis
}

// NewHealthHandler returns a new handler instance.
func NewHealthHandler(serviceName, version string, postgres *persistence.Postgres, redis *persistence.Redis) *HealthHandler {
	return &HealthHandler{serviceName: serviceName, version: version, postgres: postgres, redis: redis}
}

// Live reports service liveness.
func (h *HealthHandler) Live(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status":  "alive",
		"service": h.serviceName,
		"version": h.version,
	})
}

// Ready reports service readiness by checking dependencies.
func (h *HealthHandler) Ready(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	depStatus := fiber.Map{}
	ready := true

	if err := h.postgres.Ping(ctx); err != nil {
		depStatus["postgres"] = err.Error()
		ready = false
	} else {
		depStatus["postgres"] = "ok"
	}

	if err := h.redis.Ping(ctx); err != nil {
		depStatus["redis"] = err.Error()
		ready = false
	} else {
		depStatus["redis"] = "ok"
	}

	if ready {
		return c.JSON(fiber.Map{
			"status":       "ready",
			"dependencies": depStatus,
		})
	}

	return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
		"error": fiber.Map{
			"code":    "DEPENDENCY_UNAVAILABLE",
			"message": "one or more dependencies unavailable",
			"details": depStatus,
		},
	})
}
