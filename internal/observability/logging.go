package observability

import (
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/spec-kit/ticket-service/internal/auth"
	"github.com/spec-kit/ticket-service/internal/config"
)

// NewLogger creates a structured zap.Logger configured via env settings.
func NewLogger(cfg config.LoggerConfig) (*zap.Logger, error) {
	level := zapcore.InfoLevel
	if err := level.Set(strings.ToLower(cfg.Level)); err != nil {
		level = zapcore.InfoLevel
	}

	zapCfg := zap.Config{
		Level:       zap.NewAtomicLevelAt(level),
		Development: true,
		Encoding:    "json",
		EncoderConfig: zapcore.EncoderConfig{
			MessageKey: "message",
			LevelKey:   "level",
			TimeKey:    "ts",
			EncodeLevel: func(l zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
				enc.AppendString(l.String())
			},
			EncodeTime: zapcore.ISO8601TimeEncoder,
		},
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}

	logger, err := zapCfg.Build()
	if err != nil {
		return nil, err
	}
	return logger, nil
}

// RequestLogger logs HTTP requests with metrics integration.
func RequestLogger(logger *zap.Logger, metrics *Metrics) fiber.Handler {
	return func(c *fiber.Ctx) error {
		reqID := c.Get("X-Request-ID")
		if reqID == "" {
			reqID = uuid.NewString()
			c.Set("X-Request-ID", reqID)
		}

		start := time.Now()
		err := c.Next()
		duration := time.Since(start)
		status := c.Response().StatusCode()

		fields := []zap.Field{
			zap.String("request_id", reqID),
			zap.String("method", c.Method()),
			zap.String("path", c.Path()),
			zap.Int("status", status),
			zap.Duration("duration", duration),
		}

		if principal, ok := auth.PrincipalFromContext(c); ok {
			if principal.User != nil {
				fields = append(fields, zap.String("user_id", principal.User.ID))
			}
			if principal.Staff != nil {
				fields = append(fields,
					zap.String("staff_id", principal.Staff.ID),
					zap.String("staff_role", string(principal.Staff.Role)),
				)
			}
		}

		logger.Info("request", fields...)
		if metrics != nil {
			metrics.RecordRequest(c.Path(), c.Method(), status, duration)
		}
		return err
	}
}
