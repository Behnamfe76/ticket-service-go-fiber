package persistence

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

const migrationsDir = "migrations"

// RunMigrations executes the SQL migrations located in the /migrations directory.
func RunMigrations(ctx context.Context, pool *pgxpool.Pool, logger *zap.Logger) error {
	if pool == nil {
		logger.Warn("no postgres pool available; skipping migrations")
		return nil
	}

	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("read migrations: %w", err)
	}

	filenames := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		filenames = append(filenames, entry.Name())
	}

	sort.Strings(filenames)

	for _, name := range filenames {
		path := filepath.Join(migrationsDir, name)
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", name, err)
		}

		logger.Info("applying migration", zap.String("file", name))
		if _, err := pool.Exec(ctx, string(content)); err != nil {
			return fmt.Errorf("apply migration %s: %w", name, err)
		}
	}

	logger.Info("migrations applied", zap.Int("count", len(filenames)))
	return nil
}
