package logger

import (
	"os"

	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

func New() zerolog.Logger {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	logger := zerolog.New(os.Stdout).
		With().
		Timestamp().
		Caller().
		Logger()

	logger = logger.Level(zerolog.DebugLevel)

	return logger
}

func SetLevel(level zerolog.Level) zerolog.Logger {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	logger := zerolog.New(os.Stdout).
		With().
		Timestamp().
		Caller().
		Logger()

	logger = logger.Level(level)

	return logger
}

var Module = fx.Provide(New)
