package persistence

import (
	"context"
	"time"

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

	poolCfg, err := pgxpool.ParseConfig(cfg.DSN)
	if err != nil {
		return nil, err
	}

	if cfg.MaxConns > 0 {
		poolCfg.MaxConns = cfg.MaxConns
	}
	if cfg.MinConns > 0 {
		poolCfg.MinConns = cfg.MinConns
	}
	if cfg.ConnMaxIdleSec > 0 {
		poolCfg.MaxConnIdleTime = time.Duration(cfg.ConnMaxIdleSec) * time.Second
	}
	if cfg.ConnMaxLifeSec > 0 {
		poolCfg.MaxConnLifetime = time.Duration(cfg.ConnMaxLifeSec) * time.Second
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
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

// PoolHandle returns the underlying pgx pool.
func (p *Postgres) PoolHandle() *pgxpool.Pool {
	if p == nil {
		return nil
	}
	return p.Pool
}
