package main

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"valorant-tracker/gen/proto/valorant/v1/valorantv1connect"
	"valorant-tracker/internal/config"
	"valorant-tracker/internal/constants"
	fxmodules "valorant-tracker/internal/fx"
	"valorant-tracker/internal/middleware"
	"valorant-tracker/internal/server"

	"github.com/rs/cors"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

const ValorantTrackerPath = "/valorant.v1.ValorantTracker/"

func main() {
	fx.New(
		fxmodules.Module,
		fx.Invoke(runServer),
	).Run()
}

func runServer(
	lc fx.Lifecycle,
	trackerServer *server.TrackerServer,
	cfg *config.Config,
	db *sql.DB,
	logger zerolog.Logger,
) {
	mux := http.NewServeMux()

	_, handler := valorantv1connect.NewValorantTrackerHandler(trackerServer)

	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	})

	requestIDMiddleware := middleware.RequestID(logger)

	mux.HandleFunc(ValorantTrackerPath, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "*")
			w.WriteHeader(http.StatusNoContent)
			return
		}
		requestIDMiddleware(c.Handler(handler)).ServeHTTP(w, r)
	})

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", cfg.ServerPort),
		Handler: mux,
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go func() {
				logger.Info().Str("addr", srv.Addr).Msg("server starting")
				if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
					logger.Fatal().Err(err).Msg("server failed")
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Info().Msg("shutting down server")
			shutdownCtx, cancel := context.WithTimeout(context.Background(), constants.ShutdownTimeout)
			defer cancel()

			if err := db.Close(); err != nil {
				logger.Warn().Err(err).Msg("error closing database connection")
			}

			if err := srv.Shutdown(shutdownCtx); err != nil {
				logger.Error().Err(err).Msg("server shutdown failed")
				return err
			}
			logger.Info().Msg("server stopped gracefully")
			return nil
		},
	})
}
