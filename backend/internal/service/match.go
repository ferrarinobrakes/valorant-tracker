package service

import (
	"context"
	"fmt"
	"time"
	"valorant-tracker/internal/api"
	"valorant-tracker/internal/constants"
	"valorant-tracker/internal/domain"
	"valorant-tracker/internal/repository"

	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"
)

type MatchService struct {
	hdev           *api.HDevClient
	matchRepo      *repository.MatchRepository
	playerRepo     *repository.PlayerRepository
	mmrHistoryRepo *repository.MMRHistoryRepository
	logger         zerolog.Logger
}

func NewMatchService(hdev *api.HDevClient, matchRepo *repository.MatchRepository, playerRepo *repository.PlayerRepository, mmrHistoryRepo *repository.MMRHistoryRepository, logger zerolog.Logger) *MatchService {
	return &MatchService{hdev: hdev, matchRepo: matchRepo, playerRepo: playerRepo, mmrHistoryRepo: mmrHistoryRepo, logger: logger}
}

func (s *MatchService) GetMatchesFor(ctx context.Context, puuid string, refresh bool) ([]repository.MatchWithPlayers, error) {
	ctx, cancel := context.WithTimeout(ctx, constants.RequestTimeout)
	defer cancel()

	player, err := s.playerRepo.Get(ctx, puuid, refresh)
	if err != nil {
		s.logger.Error().Err(err).Str("puuid", puuid).Msg("player not found")
		return nil, fmt.Errorf("player not found: %w", err)
	}

	s.logger.Info().Str("player_name", player.Name).Str("player_tag", player.Tag).Str("puuid", puuid).Msg("fetching matches for player")

	storedMatches, storedMMR, err := s.fetchStoredData(ctx, player)
	if err != nil {
		s.logger.Warn().Err(err).Str("puuid", puuid).Msg("failed to fetch stored data")
	}

	if storedMatches != nil && storedMMR != nil {
		s.logger.Debug().Str("puuid", puuid).Msg("upserting stored matches")
		s.upsertStoredMatches(ctx, player.Puuid, player.Region, storedMatches.Data, storedMMR.Data, player.Name, player.Tag)
	}

	shouldRefresh, err := s.playerRepo.ShouldRefresh(ctx, player.Puuid, constants.MatchRefreshTTL)
	if err != nil {
		s.logger.Error().Err(err).Str("puuid", puuid).Msg("failed to check if matches should be refreshed")
		return nil, fmt.Errorf("failed to check if matches should be refreshed: %w", err)
	}

	s.logger.Debug().Bool("should_refresh", shouldRefresh).Str("puuid", puuid).Msg("refresh decision for matches")

	if refresh {
		s.logger.Debug().Str("puuid", puuid).Msg("manual refresh requested")
		shouldRefresh = true
	}

	if !shouldRefresh {
		s.logger.Info().Str("puuid", puuid).Msg("returning cached matches")
		return s.matchRepo.GetByPUUID(ctx, puuid)
	}

	s.logger.Info().Bool("should_refresh", shouldRefresh).Str("puuid", puuid).Msg("fetching live match data")

	v4Matches, mmrHistory, err := s.fetchLiveData(ctx, player)
	if err != nil {
		s.logger.Error().Err(err).Str("puuid", puuid).Msg("failed to fetch live data")
		return nil, fmt.Errorf("failed to fetch live data: %w", err)
	}

	s.logger.Debug().Str("puuid", puuid).Int("match_count", len(v4Matches.Data)).Msg("upserting live matches")
	s.upsertLiveMatches(ctx, player.Puuid, v4Matches.Data, mmrHistory.Data, player.Name, player.Tag)

	s.logger.Info().Str("puuid", puuid).Msg("matches fetched successfully")
	return s.matchRepo.GetByPUUID(ctx, puuid)
}

func (s *MatchService) fetchStoredData(ctx context.Context, player *domain.Player) (*api.StoredMatchesResponse, *api.StoredMMRHistoryResponse, error) {
	apiCtx, cancel := context.WithTimeout(ctx, constants.ExternalAPITimeout)
	defer cancel()

	g, gCtx := errgroup.WithContext(apiCtx)
	var storedMatches *api.StoredMatchesResponse
	var storedMMR *api.StoredMMRHistoryResponse

	hasStoredGames, err := s.matchRepo.HasStoredGames(ctx, player.Puuid)
	if err != nil {
		s.logger.Error().Err(err).Str("puuid", player.Puuid).Msg("failed to check if player has stored games")
		return nil, nil, fmt.Errorf("failed to check if player has stored games: %w", err)
	}

	if hasStoredGames {
		s.logger.Debug().Str("puuid", player.Puuid).Msg("player already has stored games, skipping fetch")
		return nil, nil, nil
	}

	g.Go(func() error {
		var err error
		storedMatches, err = s.hdev.GetStoredMatches(gCtx, player.Region, player.Puuid)
		return err
	})

	g.Go(func() error {
		var err error
		storedMMR, err = s.hdev.GetStoredMMRHistory(gCtx, player.Region, player.Puuid)
		return err
	})

	if err := g.Wait(); err != nil {
		s.logger.Error().Err(err).Str("puuid", player.Puuid).Msg("failed to fetch stored data from API")
		return nil, nil, fmt.Errorf("failed to fetch stored data: %w", err)
	}

	s.logger.Debug().Str("puuid", player.Puuid).Int("match_count", len(storedMatches.Data)).Msg("stored data fetched successfully")
	return storedMatches, storedMMR, nil
}

func (s *MatchService) fetchLiveData(ctx context.Context, player *domain.Player) (*api.V4MatchesResponse, *api.MMRHistoryResponse, error) {
	apiCtx, cancel := context.WithTimeout(ctx, constants.ExternalAPITimeout)
	defer cancel()

	g, gCtx := errgroup.WithContext(apiCtx)
	var v4Matches *api.V4MatchesResponse
	var mmrHistory *api.MMRHistoryResponse

	g.Go(func() error {
		var err error
		v4Matches, err = s.hdev.GetV4Matches(gCtx, player.Region, player.Puuid)
		return err
	})

	g.Go(func() error {
		var err error
		mmrHistory, err = s.hdev.GetMMRHistory(gCtx, player.Region, player.Puuid)
		return err
	})

	if err := g.Wait(); err != nil {
		s.logger.Error().Err(err).Str("puuid", player.Puuid).Msg("failed to fetch live data from API")
		return nil, nil, fmt.Errorf("failed to fetch live data: %w", err)
	}

	s.logger.Debug().Str("puuid", player.Puuid).Int("match_count", len(v4Matches.Data)).Msg("live data fetched successfully")
	return v4Matches, mmrHistory, nil
}

func (s *MatchService) upsertStoredMatches(ctx context.Context, puuid, region string, matches []api.StoredMatch, mmrHistory []api.StoredMMRHistoryItem, name, tag string) {
	mmrMap := make(map[string]api.StoredMMRHistoryItem)
	for _, mmr := range mmrHistory {
		mmrMap[mmr.MatchID] = mmr
	}

	var dbMatches []domain.Match
	var dbMatchPlayers []domain.MatchPlayer
	var dbMMRHistory []domain.MMRHistory

	for _, match := range matches {
		mmr, ok := mmrMap[match.Meta.ID]
		if !ok {
			continue
		}

		dbMatches = append(dbMatches, domain.Match{
			MatchID:       match.Meta.ID,
			MapName:       match.Meta.Map.Name,
			MapID:         match.Meta.Map.ID,
			Mode:          "competitive",
			StartedAt:     match.Meta.StartedAt,
			SeasonID:      match.Meta.Season.ID,
			TeamRedScore:  match.Teams.Red,
			TeamBlueScore: match.Teams.Blue,
			Region:        match.Meta.Region,
			Cluster:       match.Meta.Cluster,
			Version:       match.Meta.Version,
			Source:        "stored",
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		})

		dbMatchPlayers = append(dbMatchPlayers, domain.MatchPlayer{
			MatchID:     match.Meta.ID,
			Puuid:       puuid,
			Name:        name,
			Tag:         tag,
			Tier:        mmr.Tier.ID,
			TierName:    mmr.Tier.Name,
			Kills:       match.Stats.Kills,
			Deaths:      match.Stats.Deaths,
			Assists:     match.Stats.Assists,
			Score:       match.Stats.Score,
			Team:        match.Stats.Team,
			HasWon:      (match.Stats.Team == "Red" && match.Teams.Red > match.Teams.Blue) || (match.Stats.Team == "Blue" && match.Teams.Blue > match.Teams.Red),
			CharacterID: match.Stats.Character.ID,
			DamageTaken: match.Stats.Damage.Received,
			DamageDealt: match.Stats.Damage.Made,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		})

		dbMMRHistory = append(dbMMRHistory, domain.MMRHistory{
			MatchID:       match.Meta.ID,
			Puuid:         puuid,
			Tier:          mmr.Tier.ID,
			TierName:      mmr.Tier.Name,
			RankingInTier: mmr.RankingInTier,
			MMRChange:     mmr.LastMmrChange,
			Elo:           mmr.Elo,
			Date:          mmr.Date,
			Source:        "stored-mmr-history",
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		})
	}

	if len(dbMatches) > 0 {
		s.matchRepo.UpsertBatch(ctx, dbMatches, dbMatchPlayers)
		s.mmrHistoryRepo.UpsertBatch(ctx, dbMMRHistory)
	}
}

func (s *MatchService) upsertLiveMatches(ctx context.Context, puuid string, matches []api.V4MatchData, mmrHistory []api.MMRHistoryItem, name, tag string) {
	mmrMap := make(map[string]api.MMRHistoryItem)
	for _, mmr := range mmrHistory {
		mmrMap[mmr.MatchID] = mmr
	}

	teamWonMap := make(map[string]bool)
	for _, match := range matches {
		for _, team := range match.Teams {
			teamWonMap[team.TeamID] = team.Won
		}
	}

	var dbMatches []domain.Match
	var dbMatchPlayers []domain.MatchPlayer
	var dbMMRHistory []domain.MMRHistory

	for _, match := range matches {
		mmr, ok := mmrMap[match.Metadata.MatchID]
		if !ok {
			continue
		}

		var playerTeam string
		for _, p := range match.Players {
			if p.Puuid == puuid {
				playerTeam = p.TeamID
				break
			}
		}

		teamScoreMap := make(map[string]int)
		for _, team := range match.Teams {
			teamScoreMap[team.TeamID] = team.Rounds.Won
		}

		dbMatches = append(dbMatches, domain.Match{
			MatchID:       match.Metadata.MatchID,
			MapName:       match.Metadata.Map.Name,
			MapID:         match.Metadata.Map.ID,
			Mode:          "competitive",
			StartedAt:     match.Metadata.StartedAt,
			SeasonID:      match.Metadata.Season.ID,
			TeamRedScore:  teamScoreMap["Red"],
			TeamBlueScore: teamScoreMap["Blue"],
			Region:        match.Metadata.Region,
			Cluster:       match.Metadata.Cluster,
			Version:       match.Metadata.GameVersion,
			Source:        "v4",
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		})

		dbMatchPlayers = append(dbMatchPlayers, domain.MatchPlayer{
			MatchID:     match.Metadata.MatchID,
			Puuid:       puuid,
			Name:        name,
			Tag:         tag,
			Tier:        mmr.CurrentTier,
			TierName:    mmr.CurrentTierPatched,
			Kills:       s.getPlayerStats(match.Players, puuid, func(p api.V4Player) int { return p.Stats.Kills }),
			Deaths:      s.getPlayerStats(match.Players, puuid, func(p api.V4Player) int { return p.Stats.Deaths }),
			Assists:     s.getPlayerStats(match.Players, puuid, func(p api.V4Player) int { return p.Stats.Assists }),
			Score:       s.getPlayerStats(match.Players, puuid, func(p api.V4Player) int { return p.Stats.Score }),
			Team:        playerTeam,
			HasWon:      teamWonMap[playerTeam],
			CharacterID: s.getPlayerStatsString(match.Players, puuid, func(p api.V4Player) string { return p.Agent.ID }),
			DamageTaken: s.getPlayerStats(match.Players, puuid, func(p api.V4Player) int { return p.Stats.Damage.Received }),
			DamageDealt: s.getPlayerStats(match.Players, puuid, func(p api.V4Player) int { return p.Stats.Damage.Made }),
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		})

		dbMMRHistory = append(dbMMRHistory, domain.MMRHistory{
			MatchID:       match.Metadata.MatchID,
			Puuid:         puuid,
			Tier:          mmr.CurrentTier,
			TierName:      mmr.CurrentTierPatched,
			RankingInTier: mmr.RankingInTier,
			MMRChange:     mmr.MmrChangeToLastGame,
			Elo:           mmr.Elo,
			Date:          match.Metadata.StartedAt,
			Source:        "mmr-history",
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		})
	}

	if len(dbMatches) > 0 {
		s.matchRepo.UpsertBatch(ctx, dbMatches, dbMatchPlayers)
		s.mmrHistoryRepo.UpsertBatch(ctx, dbMMRHistory)
	}
}

func (s *MatchService) getPlayerStatsString(players []api.V4Player, targetPUUID string, getStat func(api.V4Player) string) string {
	for _, p := range players {
		if p.Puuid == targetPUUID {
			return getStat(p)
		}
	}
	return ""
}

func (s *MatchService) getPlayerStats(players []api.V4Player, targetPUUID string, getStat func(api.V4Player) int) int {
	for _, p := range players {
		if p.Puuid == targetPUUID {
			return getStat(p)
		}
	}
	return 0
}
