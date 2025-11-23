package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// Config aggregates runtime configuration for the service.
type Config struct {
	App          AppConfig
	Postgres     PostgresConfig
	Redis        RedisConfig
	Logger       LoggerConfig
	Auth         AuthConfig
	Notification NotificationConfig
}

// AppConfig controls server level behavior.
type AppConfig struct {
	Name                  string
	Env                   string
	Host                  string
	Port                  string
	Version               string
	RequestTimeoutSeconds int
}

// PostgresConfig holds DB connection values.
type PostgresConfig struct {
	DSN            string
	MaxConns       int32
	MinConns       int32
	RunMigrations  bool
	ConnMaxIdleSec int32
	ConnMaxLifeSec int32
}

// RedisConfig holds Redis connection values.
type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}

// LoggerConfig configures logging behavior.
type LoggerConfig struct {
	Level string
}

// AuthConfig defines authentication parameters.
type AuthConfig struct {
	JWTSecret               string
	AccessTokenTTLMinutes   int
	PasswordResetTTLMinutes int
	BcryptCost              int
}

// NotificationConfig holds stub notification endpoints.
type NotificationConfig struct {
	EmailFrom  string
	WebhookURL string
}

// Load reads configuration from environment variables, applying defaults where possible.
func Load() (*Config, error) {
	_ = godotenv.Load()

	redisDB, err := strconv.Atoi(getEnv("REDIS_DB", "0"))
	if err != nil {
		return nil, fmt.Errorf("invalid REDIS_DB: %w", err)
	}

	maxConns := int32(getEnvAsInt("POSTGRES_MAX_CONNS", 10))
	minConns := int32(getEnvAsInt("POSTGRES_MIN_CONNS", 2))
	runMigrations := getEnvAsBool("POSTGRES_RUN_MIGRATIONS", true)
	connMaxIdle := int32(getEnvAsInt("POSTGRES_CONN_MAX_IDLE_SECONDS", 30))
	connMaxLife := int32(getEnvAsInt("POSTGRES_CONN_MAX_LIFE_SECONDS", 300))

	cfg := &Config{
		App: AppConfig{
			Name:                  getEnv("APP_NAME", "support-ticket-service"),
			Env:                   getEnv("APP_ENV", "development"),
			Host:                  getEnv("APP_HOST", "0.0.0.0"),
			Port:                  getEnv("APP_PORT", "8080"),
			Version:               getEnv("APP_VERSION", "dev"),
			RequestTimeoutSeconds: getEnvAsInt("HTTP_REQUEST_TIMEOUT_SECONDS", 30),
		},
		Postgres: PostgresConfig{
			DSN:            os.Getenv("POSTGRES_DSN"),
			MaxConns:       maxConns,
			MinConns:       minConns,
			RunMigrations:  runMigrations,
			ConnMaxIdleSec: connMaxIdle,
			ConnMaxLifeSec: connMaxLife,
		},
		Redis: RedisConfig{
			Addr:     getEnv("REDIS_ADDR", "127.0.0.1:6379"),
			Password: os.Getenv("REDIS_PASSWORD"),
			DB:       redisDB,
		},
		Logger: LoggerConfig{
			Level: getEnv("LOG_LEVEL", "info"),
		},
		Auth: AuthConfig{
			JWTSecret:               getEnv("AUTH_JWT_SECRET", "dev-secret"),
			AccessTokenTTLMinutes:   getEnvAsInt("AUTH_ACCESS_TOKEN_TTL_MINUTES", 60),
			PasswordResetTTLMinutes: getEnvAsInt("AUTH_PASSWORD_RESET_TTL_MINUTES", 30),
			BcryptCost:              getEnvAsInt("AUTH_BCRYPT_COST", 12),
		},
		Notification: NotificationConfig{
			EmailFrom:  getEnv("NOTIFY_EMAIL_FROM", "noreply@example.com"),
			WebhookURL: getEnv("NOTIFY_WEBHOOK_URL", ""),
		},
	}

	return cfg, nil
}

// Addr returns the HTTP bind address.
func (a AppConfig) Addr() string {
	return fmt.Sprintf("%s:%s", a.Host, a.Port)
}

// RequestTimeout returns the configured request timeout duration.
func (a AppConfig) RequestTimeout() time.Duration {
	if a.RequestTimeoutSeconds <= 0 {
		return 0
	}
	return time.Duration(a.RequestTimeoutSeconds) * time.Second
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

func getEnvAsInt(key string, fallback int) int {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(val)
	if err != nil {
		return fallback
	}
	return parsed
}

func getEnvAsBool(key string, fallback bool) bool {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(val)
	if err != nil {
		return fallback
	}
	return parsed
}
