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

type MatchRepository struct {
	queries *db.Queries
	db      *sql.DB
	logger  zerolog.Logger
}

func NewMatchRepository(sqlDB *sql.DB, queries *db.Queries, logger zerolog.Logger) *MatchRepository {
	return &MatchRepository{
		queries: queries,
		db:      sqlDB,
		logger:  logger,
	}
}

// enriched
type MatchWithPlayers struct {
	Match       domain.Match
	PlayerStats domain.MatchPlayer
	MMRData     *domain.MMRHistory
}

func (r *MatchRepository) GetByPUUID(ctx context.Context, puuid string) ([]MatchWithPlayers, error) {
	matches, err := r.queries.GetMatchesByPuuid(ctx, puuid)
	if err != nil {
		return nil, err
	}

	if len(matches) == 0 {
		return []MatchWithPlayers{}, nil
	}

	matchIDs := make([]string, len(matches))
	for i, m := range matches {
		matchIDs[i] = m.MatchID
	}

	matchPlayers, err := r.queries.GetMatchPlayersByMatchIDs(ctx, db.GetMatchPlayersByMatchIDsParams{
		Puuid:    puuid,
		MatchIds: matchIDs,
	})
	if err != nil {
		return nil, err
	}

	mmrHistory, err := r.queries.GetMMRHistoryByMatchIDs(ctx, db.GetMMRHistoryByMatchIDsParams{
		Puuid:    puuid,
		MatchIds: matchIDs,
	})
	if err != nil && err != sql.ErrNoRows {
		r.logger.Warn().Err(err).Msg("failed to get MMR history")
	}

	playerStatsMap := make(map[string]db.MatchPlayer)
	for _, mp := range matchPlayers {
		playerStatsMap[mp.MatchID] = mp
	}

	mmrMap := make(map[string]db.MmrHistory)
	for _, mmr := range mmrHistory {
		mmrMap[mmr.MatchID] = mmr
	}

	results := make([]MatchWithPlayers, 0, len(matches))
	for _, match := range matches {
		playerStats := playerStatsMap[match.MatchID]
		result := MatchWithPlayers{
			Match: domain.Match{
				MatchID:       match.MatchID,
				MapName:       match.MapName,
				MapID:         match.MapID,
				Mode:          match.Mode,
				StartedAt:     match.StartedAt,
				SeasonID:      match.SeasonID,
				TeamRedScore:  int(match.TeamRedScore),
				TeamBlueScore: int(match.TeamBlueScore),
				Region:        match.Region,
				Cluster:       match.Cluster,
				Version:       match.Version,
				Source:        match.Source,
				CreatedAt:     match.CreatedAt,
				UpdatedAt:     match.UpdatedAt,
			},
			PlayerStats: domain.MatchPlayer{
				MatchID:     playerStats.MatchID,
				Puuid:       playerStats.Puuid,
				Name:        playerStats.Name,
				Tier:        int(playerStats.Tier),
				TierName:    playerStats.TierName,
				Kills:       int(playerStats.Kills),
				Deaths:      int(playerStats.Deaths),
				Assists:     int(playerStats.Assists),
				Score:       int(playerStats.Score),
				Team:        playerStats.Team,
				HasWon:      playerStats.HasWon,
				CharacterID: playerStats.CharacterID,
				DamageTaken: int(playerStats.DamageTaken),
				Tag:         playerStats.Tag,
				DamageDealt: int(playerStats.DamageDealt),
				CreatedAt:   playerStats.CreatedAt,
				UpdatedAt:   playerStats.UpdatedAt,
			},
		}

		if mmrData, hasMmr := mmrMap[match.MatchID]; hasMmr {
			result.MMRData = &domain.MMRHistory{
				ID:            mmrData.ID,
				MatchID:       mmrData.MatchID,
				Puuid:         mmrData.Puuid,
				Tier:          int(mmrData.Tier),
				TierName:      mmrData.TierName,
				RankingInTier: int(mmrData.RankingInTier),
				MMRChange:     int(mmrData.MmrChange),
				Elo:           int(mmrData.Elo),
				Date:          mmrData.Date,
				Source:        mmrData.Source,
				CreatedAt:     mmrData.CreatedAt,
				UpdatedAt:     mmrData.UpdatedAt,
			}
		}

		results = append(results, result)
	}

	return results, nil
}

func (r *MatchRepository) UpsertMatch(ctx context.Context, match *domain.Match) error {
	return r.queries.UpsertMatch(ctx, db.UpsertMatchParams{
		MatchID:       match.MatchID,
		MapName:       match.MapName,
		MapID:         match.MapID,
		Mode:          match.Mode,
		StartedAt:     match.StartedAt,
		SeasonID:      match.SeasonID,
		TeamRedScore:  int64(match.TeamRedScore),
		TeamBlueScore: int64(match.TeamBlueScore),
		Region:        match.Region,
		Cluster:       match.Cluster,
		Version:       match.Version,
		Source:        match.Source,
		CreatedAt:     match.CreatedAt,
		UpdatedAt:     match.UpdatedAt,
	})
}

func (r *MatchRepository) UpsertMatchPlayer(ctx context.Context, matchPlayer *domain.MatchPlayer) error {
	return r.queries.UpsertMatchPlayer(ctx, db.UpsertMatchPlayerParams{
		MatchID:     matchPlayer.MatchID,
		Puuid:       matchPlayer.Puuid,
		Name:        matchPlayer.Name,
		Tag:         matchPlayer.Tag,
		Tier:        int64(matchPlayer.Tier),
		TierName:    matchPlayer.TierName,
		Kills:       int64(matchPlayer.Kills),
		Deaths:      int64(matchPlayer.Deaths),
		Assists:     int64(matchPlayer.Assists),
		Score:       int64(matchPlayer.Score),
		Team:        matchPlayer.Team,
		HasWon:      matchPlayer.HasWon,
		CharacterID: matchPlayer.CharacterID,
		DamageTaken: int64(matchPlayer.DamageTaken),
		DamageDealt: int64(matchPlayer.DamageDealt),
		CreatedAt:   matchPlayer.CreatedAt,
		UpdatedAt:   matchPlayer.UpdatedAt,
	})
}

func (r *MatchRepository) UpsertBatch(ctx context.Context, matches []domain.Match, matchPlayers []domain.MatchPlayer) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	qtx := r.queries.WithTx(tx)

	if len(matches) > 0 {
		for i := 0; i < len(matches); i += constants.DBBatchSize {
			end := i + constants.DBBatchSize
			if end > len(matches) {
				end = len(matches)
			}

			for _, match := range matches[i:end] {
				err := qtx.UpsertMatch(ctx, db.UpsertMatchParams{
					MatchID:       match.MatchID,
					MapName:       match.MapName,
					MapID:         match.MapID,
					Mode:          match.Mode,
					StartedAt:     match.StartedAt,
					SeasonID:      match.SeasonID,
					TeamRedScore:  int64(match.TeamRedScore),
					TeamBlueScore: int64(match.TeamBlueScore),
					Region:        match.Region,
					Cluster:       match.Cluster,
					Version:       match.Version,
					Source:        match.Source,
					CreatedAt:     match.CreatedAt,
					UpdatedAt:     match.UpdatedAt,
				})
				if err != nil {
					return fmt.Errorf("failed to upsert match %s: %w", match.MatchID, err)
				}
			}
		}
	}

	if len(matchPlayers) > 0 {
		for i := 0; i < len(matchPlayers); i += constants.DBBatchSize {
			end := i + constants.DBBatchSize
			if end > len(matchPlayers) {
				end = len(matchPlayers)
			}

			for _, mp := range matchPlayers[i:end] {
				err := qtx.UpsertMatchPlayer(ctx, db.UpsertMatchPlayerParams{
					MatchID:     mp.MatchID,
					Puuid:       mp.Puuid,
					Name:        mp.Name,
					Tag:         mp.Tag,
					Tier:        int64(mp.Tier),
					TierName:    mp.TierName,
					Kills:       int64(mp.Kills),
					Deaths:      int64(mp.Deaths),
					Assists:     int64(mp.Assists),
					Score:       int64(mp.Score),
					Team:        mp.Team,
					HasWon:      mp.HasWon,
					CharacterID: mp.CharacterID,
					DamageTaken: int64(mp.DamageTaken),
					DamageDealt: int64(mp.DamageDealt),
					CreatedAt:   mp.CreatedAt,
					UpdatedAt:   mp.UpdatedAt,
				})
				if err != nil {
					return fmt.Errorf("failed to upsert match player %s/%s: %w", mp.MatchID, mp.Puuid, err)
				}
			}
		}
	}

	return tx.Commit()
}

func (r *MatchRepository) GetLatestMatchDate(ctx context.Context, puuid string) (*time.Time, error) {
	startedAt, err := r.queries.GetLatestMatchDate(ctx, puuid)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &startedAt, nil
}

func (r *MatchRepository) HasStoredGames(ctx context.Context, puuid string) (bool, error) {
	count, err := r.queries.CountStoredGames(ctx, puuid)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *MatchRepository) GetByMatchID(ctx context.Context, matchID string) ([]domain.MatchPlayer, error) {
	players, err := r.queries.GetMatchPlayersByMatchID(ctx, matchID)
	if err != nil {
		return nil, err
	}

	result := make([]domain.MatchPlayer, len(players))
	for i, p := range players {
		result[i] = domain.MatchPlayer{
			MatchID:     p.MatchID,
			Puuid:       p.Puuid,
			Name:        p.Name,
			Tier:        int(p.Tier),
			TierName:    p.TierName,
			Kills:       int(p.Kills),
			Deaths:      int(p.Deaths),
			Assists:     int(p.Assists),
			Score:       int(p.Score),
			Team:        p.Team,
			HasWon:      p.HasWon,
			CharacterID: p.CharacterID,
			DamageTaken: int(p.DamageTaken),
			Tag:         p.Tag,
			DamageDealt: int(p.DamageDealt),
			CreatedAt:   p.CreatedAt,
			UpdatedAt:   p.UpdatedAt,
		}
	}
	return result, nil
}

func (r *MatchRepository) GetMatchMetadata(ctx context.Context, matchID string) (*domain.Match, error) {
	match, err := r.queries.GetMatchMetadata(ctx, matchID)
	if err != nil {
		return nil, err
	}

	return &domain.Match{
		MatchID:       match.MatchID,
		MapName:       match.MapName,
		MapID:         match.MapID,
		Mode:          match.Mode,
		StartedAt:     match.StartedAt,
		SeasonID:      match.SeasonID,
		TeamRedScore:  int(match.TeamRedScore),
		TeamBlueScore: int(match.TeamBlueScore),
		Region:        match.Region,
		Cluster:       match.Cluster,
		Version:       match.Version,
		Source:        match.Source,
		CreatedAt:     match.CreatedAt,
		UpdatedAt:     match.UpdatedAt,
	}, nil
}
