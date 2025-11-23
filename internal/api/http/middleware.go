package http

import (
	"context"
	"runtime/debug"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"

	"github.com/spec-kit/ticket-service/internal/observability"
	apperrors "github.com/spec-kit/ticket-service/pkg/util/errorutil"
)

// RegisterMiddlewares attaches global middlewares such as error handling and logging.
func RegisterMiddlewares(app *fiber.App, logger *zap.Logger, metrics *observability.Metrics, timeout time.Duration) {
	if timeout > 0 {
		app.Use(requestTimeoutMiddleware(timeout))
	}
	app.Use(errorHandlingMiddleware(logger, metrics))
	app.Use(observability.RequestLogger(logger, metrics))
}

func requestTimeoutMiddleware(timeout time.Duration) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx, cancel := context.WithTimeout(c.UserContext(), timeout)
		defer cancel()
		c.SetUserContext(ctx)
		return c.Next()
	}
}

func errorHandlingMiddleware(logger *zap.Logger, metrics *observability.Metrics) fiber.Handler {
	return func(c *fiber.Ctx) (err error) {
		defer func() {
			if r := recover(); r != nil {
				logger.Error("panic recovered", zap.Any("panic", r), zap.ByteString("stack", debug.Stack()))
				err = apperrors.NewInternalError(nil)
			}
			if err != nil {
				domainErr := apperrors.ToDomainError(err)
				if metrics != nil {
					metrics.RecordError(c.Path(), c.Method(), domainErr.Code)
				}
				response := fiber.Map{"error": fiber.Map{
					"code":    domainErr.Code,
					"message": domainErr.Message,
				}}
				if len(domainErr.Details) > 0 {
					response["error"].(fiber.Map)["details"] = domainErr.Details
				}
				if domainErr.HTTPStatus >= 500 {
					logger.Error("request failed", zap.Error(domainErr))
				}
				c.Status(domainErr.HTTPStatus)
				_ = c.JSON(response)
				err = nil
			}
		}()
		return c.Next()
	}
}
