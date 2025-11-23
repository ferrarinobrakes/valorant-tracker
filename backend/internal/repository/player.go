package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"
	"valorant-tracker/internal/constants"
	"valorant-tracker/internal/db"
	"valorant-tracker/internal/domain"

	"github.com/rs/zerolog"
)

type PlayerRepository struct {
	queries *db.Queries
	db      *sql.DB
	logger  zerolog.Logger
}

func NewPlayerRepository(sqlDB *sql.DB, queries *db.Queries, logger zerolog.Logger) *PlayerRepository {
	return &PlayerRepository{
		queries: queries,
		db:      sqlDB,
		logger:  logger,
	}
}

func (r *PlayerRepository) Get(ctx context.Context, puuid string, refresh bool) (*domain.Player, error) {
	player, err := r.queries.GetPlayerByPuuid(ctx, puuid)
	if err != nil {
		return nil, err
	}

	return &domain.Player{
		Puuid:           player.Puuid,
		Name:            player.Name,
		Tag:             player.Tag,
		Region:          player.Region,
		AccountLevel:    int(player.AccountLevel),
		Card:            player.Card,
		Title:           player.Title,
		CurrentTier:     int(player.CurrentTier),
		CurrentTierName: player.CurrentTierName,
		CurrentRR:       int(player.CurrentRr),
		IsPartialFetch:  player.IsPartialFetch,
		LastFetchAt:     player.LastFetchAt,
		CreatedAt:       player.CreatedAt,
		UpdatedAt:       player.UpdatedAt,
	}, nil
}

func (r *PlayerRepository) Upsert(ctx context.Context, player *domain.Player) error {
	return r.queries.UpsertPlayer(ctx, db.UpsertPlayerParams{
		Puuid:           player.Puuid,
		Name:            player.Name,
		Tag:             player.Tag,
		Region:          player.Region,
		AccountLevel:    int64(player.AccountLevel),
		Card:            player.Card,
		Title:           player.Title,
		CurrentTier:     int64(player.CurrentTier),
		CurrentTierName: player.CurrentTierName,
		CurrentRr:       int64(player.CurrentRR),
		IsPartialFetch:  player.IsPartialFetch,
		LastFetchAt:     player.LastFetchAt,
		CreatedAt:       player.CreatedAt,
		UpdatedAt:       player.UpdatedAt,
	})
}

func (r *PlayerRepository) ShouldRefresh(ctx context.Context, puuid string, ttl time.Duration) (bool, error) {
	player, err := r.queries.GetPlayerLastFetchAt(ctx, puuid)
	if err == sql.ErrNoRows {
		r.logger.Debug().Str("puuid", puuid).Msg("player not found, should refresh")
		return true, nil
	}
	if err != nil {
		r.logger.Error().Err(err).Str("puuid", puuid).Msg("failed to get player")
		return false, err
	}
	if player.IsPartialFetch {
		r.logger.Debug().Str("puuid", puuid).Msg("player is partial fetch, should refresh")
		return true, nil
	}

	timeSince := time.Since(player.LastFetchAt)
	shouldRefresh := timeSince > ttl
	r.logger.Debug().
		Str("puuid", puuid).
		Time("last_fetch_at", player.LastFetchAt).
		Dur("time_since", timeSince).
		Dur("ttl", ttl).
		Bool("should_refresh", shouldRefresh).
		Msg("checking if player should refresh")

	return shouldRefresh, nil
}

func (r *PlayerRepository) SetLastFetchAt(puuid string, lastFetchAt time.Time) error {
	r.logger.Debug().
		Str("puuid", puuid).
		Time("last_fetch_at", lastFetchAt).
		Msg("setting last fetch at")

	err := r.queries.UpdatePlayerLastFetchAt(context.Background(), db.UpdatePlayerLastFetchAtParams{
		LastFetchAt: lastFetchAt,
		UpdatedAt:   time.Now(),
		Puuid:       puuid,
	})

	if err != nil {
		r.logger.Error().Err(err).Str("puuid", puuid).Msg("failed to set last fetch at")
		return err
	}

	r.logger.Debug().Str("puuid", puuid).Msg("last fetch at set successfully")
	return nil
}

func (r *PlayerRepository) UpsertBatch(ctx context.Context, players []domain.Player) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	qtx := r.queries.WithTx(tx)

	for i := 0; i < len(players); i += constants.DBBatchSize {
		end := i + constants.DBBatchSize
		if end > len(players) {
			end = len(players)
		}

		for _, player := range players[i:end] {
			err := qtx.UpsertPlayer(ctx, db.UpsertPlayerParams{
				Puuid:           player.Puuid,
				Name:            player.Name,
				Tag:             player.Tag,
				Region:          player.Region,
				AccountLevel:    int64(player.AccountLevel),
				Card:            player.Card,
				Title:           player.Title,
				CurrentTier:     int64(player.CurrentTier),
				CurrentTierName: player.CurrentTierName,
				CurrentRr:       int64(player.CurrentRR),
				IsPartialFetch:  player.IsPartialFetch,
				LastFetchAt:     player.LastFetchAt,
				CreatedAt:       player.CreatedAt,
				UpdatedAt:       player.UpdatedAt,
			})
			if err != nil {
				return fmt.Errorf("failed to upsert player %s: %w", player.Puuid, err)
			}
		}
	}

	return tx.Commit()
}

func (r *PlayerRepository) Search(ctx context.Context, query string, limit int) ([]domain.Player, error) {
	searchPattern := "%" + query + "%"
	players, err := r.queries.SearchPlayers(ctx, db.SearchPlayersParams{
		Name:  searchPattern,
		Tag:   searchPattern,
		Limit: int64(limit),
	})
	if err != nil {
		return nil, err
	}

	result := make([]domain.Player, len(players))
	for i, p := range players {
		result[i] = domain.Player{
			Puuid:           p.Puuid,
			Name:            p.Name,
			Tag:             p.Tag,
			Region:          p.Region,
			AccountLevel:    int(p.AccountLevel),
			Card:            p.Card,
			Title:           p.Title,
			CurrentTier:     int(p.CurrentTier),
			CurrentTierName: p.CurrentTierName,
			CurrentRR:       int(p.CurrentRr),
			IsPartialFetch:  p.IsPartialFetch,
			LastFetchAt:     p.LastFetchAt,
			CreatedAt:       p.CreatedAt,
			UpdatedAt:       p.UpdatedAt,
		}
	}
	return result, nil
}

func (r *PlayerRepository) GetByName(ctx context.Context, name, tag string) (*domain.Player, error) {
	player, err := r.queries.GetPlayerByNameTag(ctx, db.GetPlayerByNameTagParams{
		Name: name,
		Tag:  tag,
	})
	if err != nil {
		return nil, err
	}

	return &domain.Player{
		Puuid:           player.Puuid,
		Name:            player.Name,
		Tag:             player.Tag,
		Region:          player.Region,
		AccountLevel:    int(player.AccountLevel),
		Card:            player.Card,
		Title:           player.Title,
		CurrentTier:     int(player.CurrentTier),
		CurrentTierName: player.CurrentTierName,
		CurrentRR:       int(player.CurrentRr),
		IsPartialFetch:  player.IsPartialFetch,
		LastFetchAt:     player.LastFetchAt,
		CreatedAt:       player.CreatedAt,
		UpdatedAt:       player.UpdatedAt,
	}, nil
}

func (r *PlayerRepository) SetPartialFetch(ctx context.Context, puuid string, isPartialFetch bool) error {
	return r.queries.UpdatePlayerPartialFetch(ctx, db.UpdatePlayerPartialFetchParams{
		IsPartialFetch: isPartialFetch,
		UpdatedAt:      time.Now(),
		Puuid:          puuid,
	})
}
