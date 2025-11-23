package config

import (
	"fmt"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

type Config struct {
	HDevAPIKey string
	DBPath     string
	ServerPort string
	LogLevel   string
	CacheTTL   time.Duration
}

func Load(logger zerolog.Logger) (*Config, error) {
	if err := godotenv.Load(); err != nil {
		logger.Debug().Msg(".env file not found, using environment variables or defaults")
	}

	cfg := &Config{
		HDevAPIKey: getEnv("HDEV_API_KEY", ""),
		DBPath:     getEnv("DB_PATH", "valorant.db"),
		ServerPort: getEnv("SERVER_PORT", "8080"),
		LogLevel:   getEnv("LOG_LEVEL", "info"),
		CacheTTL:   5 * time.Minute,
	}

	if cfg.HDevAPIKey == "" {
		return nil, fmt.Errorf("HDEV_API_KEY is required")
	}

	logger.Info().
		Str("db_path", cfg.DBPath).
		Str("server_port", cfg.ServerPort).
		Str("log_level", cfg.LogLevel).
		Dur("cache_ttl", cfg.CacheTTL).
		Msg("configuration loaded")

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

var Module = fx.Provide(Load)
