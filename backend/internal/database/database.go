package database

import (
	"database/sql"
	"embed"
	"fmt"
	"valorant-tracker/internal/config"
	"valorant-tracker/internal/constants"

	_ "github.com/mattn/go-sqlite3"
	"github.com/pressly/goose/v3"
	"github.com/rs/zerolog"
)

//go:embed migrations/*.sql
var embedMigrations embed.FS

func New(cfg *config.Config, logger zerolog.Logger) (*sql.DB, error) {
	logger.Info().Str("path", cfg.DBPath).Msg("connecting to database")

	db, err := sql.Open("sqlite3", cfg.DBPath)
	if err != nil {
		logger.Error().Err(err).Msg("failed to connect to database")
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	db.SetMaxOpenConns(constants.DBMaxOpenConns)
	db.SetMaxIdleConns(constants.DBMaxIdleConns)
	db.SetConnMaxLifetime(constants.DBConnMaxLifetime)
	db.SetConnMaxIdleTime(constants.DBMaxIdleTime)

	if err := optimizeSQLite(db, logger); err != nil {
		logger.Error().Err(err).Msg("failed to optimize SQLite")
		return nil, fmt.Errorf("failed to optimize SQLite: %w", err)
	}
	if err := runMigrations(db, logger); err != nil {
		logger.Error().Err(err).Msg("failed to run migrations")
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	logger.Info().Msg("database connection established and optimized")
	return db, nil
}

func runMigrations(db *sql.DB, logger zerolog.Logger) error {
	goose.SetBaseFS(embedMigrations)

	if err := goose.SetDialect("sqlite3"); err != nil {
		return fmt.Errorf("failed to set goose dialect: %w", err)
	}

	if err := goose.Up(db, "migrations"); err != nil {
		return fmt.Errorf("failed to run goose migrations: %w", err)
	}

	logger.Info().Msg("migrations completed successfully")
	return nil
}

func optimizeSQLite(sqlDB *sql.DB, logger zerolog.Logger) error {
	pragmas := []struct {
		name  string
		value string
	}{
		{"journal_mode", "WAL"},
		{"synchronous", "NORMAL"},
		{"cache_size", "-64000"},
		{"busy_timeout", "5000"},
		{"foreign_keys", "ON"},
		{"temp_store", "MEMORY"},
		{"mmap_size", "268435456"}, // memory map 256MB for better performance https://sqlite.org/mmap.html
	}

	for _, pragma := range pragmas {
		query := fmt.Sprintf("PRAGMA %s = %s", pragma.name, pragma.value)
		if _, err := sqlDB.Exec(query); err != nil {
			logger.Warn().
				Err(err).
				Str("pragma", pragma.name).
				Str("value", pragma.value).
				Msg("failed to set pragma")
			return fmt.Errorf("failed to set PRAGMA %s: %w", pragma.name, err)
		}
		logger.Debug().
			Str("pragma", pragma.name).
			Str("value", pragma.value).
			Msg("SQLite pragma set")
	}

	return nil
}
