package repository

import (
	"context"
	"database/sql"
	"fmt"
	"valorant-tracker/internal/db"
	"valorant-tracker/internal/domain"

	gonanoid "github.com/matoous/go-nanoid/v2"
	"github.com/rs/zerolog"
)

type MMRHistoryRepository struct {
	queries *db.Queries
	db      *sql.DB
	logger  zerolog.Logger
}

func NewMMRHistoryRepository(sqlDB *sql.DB, queries *db.Queries, logger zerolog.Logger) *MMRHistoryRepository {
	return &MMRHistoryRepository{
		queries: queries,
		db:      sqlDB,
		logger:  logger,
	}
}

func (r *MMRHistoryRepository) UpsertBatch(ctx context.Context, records []domain.MMRHistory) error {
	if len(records) == 0 {
		return nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	qtx := r.queries.WithTx(tx)

	for _, record := range records {
		id := record.ID
		if id == "" {
			id, err = gonanoid.New()
			if err != nil {
				return fmt.Errorf("failed to generate nanoid: %w", err)
			}
		}

		err := qtx.UpsertMMRHistory(ctx, db.UpsertMMRHistoryParams{
			ID:            id,
			MatchID:       record.MatchID,
			Puuid:         record.Puuid,
			Tier:          int64(record.Tier),
			TierName:      record.TierName,
			RankingInTier: int64(record.RankingInTier),
			MmrChange:     int64(record.MMRChange),
			Elo:           int64(record.Elo),
			Date:          record.Date,
			Source:        record.Source,
			CreatedAt:     record.CreatedAt,
			UpdatedAt:     record.UpdatedAt,
		})
		if err != nil {
			return fmt.Errorf("failed to upsert mmr history: %w", err)
		}
	}

	return tx.Commit()
}

func (r *MMRHistoryRepository) GetByPuuid(ctx context.Context, puuid string, limit int) ([]domain.MMRHistory, error) {
	records, err := r.queries.GetMMRHistoryByPuuid(ctx, db.GetMMRHistoryByPuuidParams{
		Puuid: puuid,
		Limit: int64(limit),
	})
	if err != nil {
		return nil, err
	}

	result := make([]domain.MMRHistory, len(records))
	for i, r := range records {
		result[i] = domain.MMRHistory{
			ID:            r.ID,
			MatchID:       r.MatchID,
			Puuid:         r.Puuid,
			Tier:          int(r.Tier),
			TierName:      r.TierName,
			RankingInTier: int(r.RankingInTier),
			MMRChange:     int(r.MmrChange),
			Elo:           int(r.Elo),
			Date:          r.Date,
			Source:        r.Source,
			CreatedAt:     r.CreatedAt,
			UpdatedAt:     r.UpdatedAt,
		}
	}
	return result, nil
}
