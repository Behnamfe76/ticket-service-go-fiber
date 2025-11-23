package persistence

import (
	"context"
	"errors"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/spec-kit/ticket-service/internal/config"
)

// Redis wraps the go-redis client.
type Redis struct {
	Client *redis.Client
}

// NewRedis connects to Redis using the provided configuration.
func NewRedis(cfg config.RedisConfig, logger *zap.Logger) *Redis {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	if err := client.Ping(context.Background()).Err(); err != nil {
		logger.Warn("unable to reach redis", zap.Error(err))
	} else {
		logger.Info("connected to redis")
	}

	return &Redis{Client: client}
}

// Close closes the client.
func (r *Redis) Close() {
	if r != nil && r.Client != nil {
		_ = r.Client.Close()
	}
}

// Ping verifies Redis connectivity.
func (r *Redis) Ping(ctx context.Context) error {
	if r == nil || r.Client == nil {
		return errors.New("redis client not configured")
	}
	return r.Client.Ping(ctx).Err()
}
