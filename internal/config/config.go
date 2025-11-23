package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config aggregates runtime configuration for the service.
type Config struct {
	App      AppConfig
	Postgres PostgresConfig
	Redis    RedisConfig
	Logger   LoggerConfig
}

// AppConfig controls server level behavior.
type AppConfig struct {
	Name string
	Env  string
	Host string
	Port string
}

// PostgresConfig holds DB connection values.
type PostgresConfig struct {
	DSN string
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

// Load reads configuration from environment variables, applying defaults where possible.
func Load() (*Config, error) {
	_ = godotenv.Load()

	redisDB, err := strconv.Atoi(getEnv("REDIS_DB", "0"))
	if err != nil {
		return nil, fmt.Errorf("invalid REDIS_DB: %w", err)
	}

	cfg := &Config{
		App: AppConfig{
			Name: getEnv("APP_NAME", "support-ticket-service"),
			Env:  getEnv("APP_ENV", "development"),
			Host: getEnv("APP_HOST", "0.0.0.0"),
			Port: getEnv("APP_PORT", "8080"),
		},
		Postgres: PostgresConfig{
			DSN: os.Getenv("POSTGRES_DSN"),
		},
		Redis: RedisConfig{
			Addr:     getEnv("REDIS_ADDR", "127.0.0.1:6379"),
			Password: os.Getenv("REDIS_PASSWORD"),
			DB:       redisDB,
		},
		Logger: LoggerConfig{
			Level: getEnv("LOG_LEVEL", "info"),
		},
	}

	return cfg, nil
}

// Addr returns the HTTP bind address.
func (a AppConfig) Addr() string {
	return fmt.Sprintf("%s:%s", a.Host, a.Port)
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
