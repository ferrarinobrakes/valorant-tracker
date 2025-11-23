package fx

import (
	"database/sql"
	"valorant-tracker/internal/api"
	"valorant-tracker/internal/config"
	"valorant-tracker/internal/database"
	"valorant-tracker/internal/db"
	"valorant-tracker/internal/logger"
	"valorant-tracker/internal/repository"
	"valorant-tracker/internal/server"
	"valorant-tracker/internal/service"

	"go.uber.org/fx"
)

func ProvideQueries(sqlDB *sql.DB) *db.Queries {
	return db.New(sqlDB)
}

var Module = fx.Options(
	fx.Provide(logger.New),
	fx.Provide(config.Load),
	fx.Provide(database.New),
	fx.Provide(ProvideQueries),
	// repos
	fx.Provide(repository.NewPlayerRepository),
	fx.Provide(repository.NewMatchRepository),
	fx.Provide(repository.NewMMRHistoryRepository),
	// api client
	fx.Provide(api.NewHDevClient),
	// svc
	fx.Provide(service.NewPlayerService),
	fx.Provide(service.NewMatchService),
	fx.Provide(service.NewMatchDetailService),
	// server
	fx.Provide(server.NewTrackerServer),
)
