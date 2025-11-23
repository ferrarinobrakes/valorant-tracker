package service

import (
	"context"
	"fmt"
	"net/url"
	"time"
	valorantv1 "valorant-tracker/gen/proto/valorant/v1"
	"valorant-tracker/internal/api"
	"valorant-tracker/internal/constants"
	"valorant-tracker/internal/domain"
	"valorant-tracker/internal/repository"

	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"
)

type PlayerService struct {
	hdev   *api.HDevClient
	repo   *repository.PlayerRepository
	logger zerolog.Logger
}

func NewPlayerService(hdev *api.HDevClient, repo *repository.PlayerRepository, logger zerolog.Logger) *PlayerService {
	return &PlayerService{hdev: hdev, repo: repo, logger: logger}
}

func (s *PlayerService) GetPlayer(ctx context.Context, name, tag string, refresh bool) (*domain.Player, error) {
	ctx, cancel := context.WithTimeout(ctx, constants.RequestTimeout)
	defer cancel()

	name, err := url.QueryUnescape(name)
	if err != nil {
		return nil, fmt.Errorf("failed to unescape name: %w", err)
	}
	tag, err = url.QueryUnescape(tag)
	if err != nil {
		return nil, fmt.Errorf("failed to unescape tag: %w", err)
	}

	s.logger.Info().Str("name", name).Str("tag", tag).Bool("refresh", refresh).Msg("getting player")

	var exists bool
	var shouldRefresh bool

	player, err := s.repo.GetByName(ctx, name, tag)
	if err == nil && player != nil {
		exists = true

		shouldRefresh, err = s.repo.ShouldRefresh(ctx, player.Puuid, constants.PlayerRefreshTTL)
		if err != nil {
			return nil, err
		}

		if player.IsPartialFetch {
			shouldRefresh = true
			s.logger.Debug().Str("puuid", player.Puuid).Msg("player is partial fetch, forcing refresh")
		}

		if refresh {
			shouldRefresh = true
			s.logger.Debug().Str("puuid", player.Puuid).Msg("manual refresh requested")
		}

		s.logger.Debug().
			Bool("shouldRefresh", shouldRefresh).
			Bool("exists", exists).
			Bool("isPartialFetch", player.IsPartialFetch).
			Msg("refresh decision")

		if !shouldRefresh {
			player, err := s.repo.Get(ctx, player.Puuid, shouldRefresh)
			if err == nil {
				s.logger.Info().Str("puuid", player.Puuid).Msg("returning cached player")
				return player, nil
			}
		}
	} else {
		shouldRefresh = true
		s.logger.Debug().Str("name", name).Str("tag", tag).Msg("player not found in database, fetching from API")
	}

	apiCtx, apiCancel := context.WithTimeout(ctx, constants.ExternalAPITimeout)
	defer apiCancel()

	accResponse, err := s.hdev.GetAccount(apiCtx, name, tag)
	if err != nil {
		s.logger.Error().Err(err).Str("name", name).Str("tag", tag).Msg("failed to fetch account")
		return nil, fmt.Errorf("failed to fetch account: %w", err)
	}

	player = &domain.Player{
		Puuid:        accResponse.Data.Puuid,
		Name:         accResponse.Data.Name,
		Tag:          accResponse.Data.Tag,
		Region:       accResponse.Data.Region,
		AccountLevel: accResponse.Data.AccountLevel,
		Card:         accResponse.Data.Card,
		Title:        accResponse.Data.Title,
	}

	player.IsPartialFetch = false
	if err := s.repo.SetPartialFetch(ctx, player.Puuid, player.IsPartialFetch); err != nil {
		s.logger.Warn().Err(err).Str("puuid", player.Puuid).Msg("failed to set partial fetch")
		return nil, fmt.Errorf("failed to set partial fetch: %w", err)
	}

	var mmr *api.MMRResponse
	if shouldRefresh {
		mmrCtx, mmrCancel := context.WithTimeout(ctx, constants.ExternalAPITimeout)
		defer mmrCancel()

		mmr, err = s.hdev.GetMMR(mmrCtx, player.Region, player.Puuid)
		if err != nil {
			s.logger.Error().Err(err).Str("puuid", player.Puuid).Msg("failed to fetch mmr")
			return nil, fmt.Errorf("failed to fetch mmr: %w", err)
		}
	} else {
		mmr = &api.MMRResponse{
			Data: api.MMRCurrent{
				Current: struct {
					Tier struct {
						ID   int    `json:"id"`
						Name string `json:"name"`
					} `json:"tier"`
					RR int `json:"rr"`
				}{
					Tier: struct {
						ID   int    `json:"id"`
						Name string `json:"name"`
					}{
						ID:   player.CurrentTier,
						Name: player.CurrentTierName,
					},
					RR: player.CurrentRR,
				},
			},
		}
	}

	player = &domain.Player{
		Puuid:           player.Puuid,
		Name:            player.Name,
		Tag:             player.Tag,
		Region:          player.Region,
		AccountLevel:    player.AccountLevel,
		Card:            player.Card,
		Title:           player.Title,
		CurrentTier:     mmr.Data.Current.Tier.ID,
		CurrentTierName: mmr.Data.Current.Tier.Name,
		CurrentRR:       mmr.Data.Current.RR,
	}

	if err := s.repo.Upsert(ctx, player); err != nil {
		s.logger.Error().Err(err).Str("puuid", player.Puuid).Msg("failed to upsert player")
		return nil, fmt.Errorf("failed to upsert player: %w", err)
	}

	g := new(errgroup.Group)
	g.Go(func() error {
		time.Sleep(constants.LastFetchDelay)
		s.logger.Debug().Str("puuid", player.Puuid).Msg("setting last fetch at")
		if err := s.repo.SetLastFetchAt(player.Puuid, time.Now()); err != nil {
			s.logger.Warn().Err(err).Str("puuid", player.Puuid).Msg("failed to set last fetch at")
			return err
		}
		return nil
	})

	go func() {
		if err := g.Wait(); err != nil {
			s.logger.Error().Err(err).Msg("background task failed")
		}
	}()

	s.logger.Info().Str("puuid", player.Puuid).Msg("player fetched successfully")
	return player, nil
}

func (s *PlayerService) SearchSuggestions(ctx context.Context, query string) ([]*valorantv1.PlayerResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, constants.DatabaseTimeout)
	defer cancel()

	s.logger.Debug().Str("query", query).Msg("searching players")

	players, err := s.repo.Search(ctx, query, constants.SearchSuggestionLimit)
	if err != nil {
		s.logger.Error().Err(err).Str("query", query).Msg("failed to search players")
		return nil, err
	}

	var suggestions []*valorantv1.PlayerResponse
	for _, p := range players {
		suggestions = append(suggestions, &valorantv1.PlayerResponse{
			Puuid:        p.Puuid,
			Name:         p.Name,
			Tag:          p.Tag,
			Region:       p.Region,
			AccountLevel: int32(p.AccountLevel),
			Card:         p.Card,
			Title:        p.Title,
			CurrentTier: &valorantv1.Tier{
				Id:   int32(p.CurrentTier),
				Name: p.CurrentTierName,
			},
			CurrentRr: int32(p.CurrentRR),
		})
	}

	s.logger.Info().Int("count", len(suggestions)).Str("query", query).Msg("search completed")
	return suggestions, nil
}

func (s *PlayerService) GetPlayerByPuuid(ctx context.Context, puuid string, refresh bool) (*domain.Player, error) {
	ctx, cancel := context.WithTimeout(ctx, constants.DatabaseTimeout)
	defer cancel()

	s.logger.Debug().Str("puuid", puuid).Bool("refresh", refresh).Msg("getting player by puuid")

	player, err := s.repo.Get(ctx, puuid, refresh)
	if err != nil {
		s.logger.Error().Err(err).Str("puuid", puuid).Msg("player not found")
		return nil, err
	}

	return player, nil
}
