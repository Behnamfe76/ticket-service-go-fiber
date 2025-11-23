package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"

	httptransport "github.com/spec-kit/ticket-service/internal/api/http"
	"github.com/spec-kit/ticket-service/internal/api/http/handlers"
	"github.com/spec-kit/ticket-service/internal/auth"
	"github.com/spec-kit/ticket-service/internal/config"
	"github.com/spec-kit/ticket-service/internal/observability"
	"github.com/spec-kit/ticket-service/internal/persistence"
	"github.com/spec-kit/ticket-service/internal/repository"
	"github.com/spec-kit/ticket-service/internal/service"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	logger, err := observability.NewLogger(cfg.Logger)
	if err != nil {
		log.Fatalf("failed to init logger: %v", err)
	}
	defer logger.Sync() //nolint:errcheck

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pg, err := persistence.NewPostgres(ctx, cfg.Postgres, logger)
	if err != nil {
		logger.Fatal("failed to connect postgres", zap.Error(err))
	}
	defer pg.Close()

	if cfg.Postgres.RunMigrations {
		if err := persistence.RunMigrations(ctx, pg.PoolHandle(), logger); err != nil {
			logger.Fatal("failed to run migrations", zap.Error(err))
		}
	}

	redis := persistence.NewRedis(cfg.Redis, logger)
	defer redis.Close()

	pool := pg.PoolHandle()
	userRepo := repository.NewUserRepository(pool)
	staffRepo := repository.NewStaffRepository(pool)
	resetRepo := repository.NewPasswordResetRepository(pool)

	authService := service.NewAuthService(*cfg, service.AuthDependencies{
		UserRepo:          userRepo,
		StaffRepo:         staffRepo,
		PasswordResetRepo: resetRepo,
	})
	authMiddleware := auth.NewAuthMiddleware(authService.TokenManager(), userRepo, staffRepo)

	app := fiber.New()
	httptransport.RegisterMiddlewares(app, logger)

	healthHandler := handlers.NewHealthHandler()
	usersHandler := handlers.NewUsersHandler(authService)
	staffHandler := handlers.NewStaffHandler(authService)

	httptransport.RegisterRoutes(app, httptransport.RouteConfig{
		Health:         healthHandler,
		Users:          usersHandler,
		Staff:          staffHandler,
		AuthMiddleware: authMiddleware,
	})

	go func() {
		if err := app.Listen(cfg.App.Addr()); err != nil {
			logger.Fatal("fiber listen", zap.Error(err))
		}
	}()

	waitForShutdown(logger)

	_ = app.Shutdown()
}

func waitForShutdown(logger *zap.Logger) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	sig := <-sigCh
	logger.Info("shutting down", zap.String("signal", sig.String()))
}
