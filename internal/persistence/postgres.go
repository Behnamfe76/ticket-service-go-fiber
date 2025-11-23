package persistence

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"github.com/spec-kit/ticket-service/internal/config"
)

// Postgres wraps access to a pgx connection pool.
type Postgres struct {
	Pool *pgxpool.Pool
}

// NewPostgres establishes a connection pool when DSN is provided.
func NewPostgres(ctx context.Context, cfg config.PostgresConfig, logger *zap.Logger) (*Postgres, error) {
	if cfg.DSN == "" {
		logger.Warn("POSTGRES_DSN not provided; skipping database connection")
		return &Postgres{Pool: nil}, nil
	}

	pool, err := pgxpool.New(ctx, cfg.DSN)
	if err != nil {
		return nil, err
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}

	logger.Info("connected to postgres")
	return &Postgres{Pool: pool}, nil
}

// Close releases pool resources.
func (p *Postgres) Close() {
	if p != nil && p.Pool != nil {
		p.Pool.Close()
	}
}
