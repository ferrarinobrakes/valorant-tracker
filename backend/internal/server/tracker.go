package server

import (
	"context"
	"fmt"
	"time"
	valorantv1 "valorant-tracker/gen/proto/valorant/v1"
	"valorant-tracker/internal/domain"
	"valorant-tracker/internal/repository"
	"valorant-tracker/internal/service"

	"connectrpc.com/connect"
)

type TrackerServer struct {
	playerSvc      *service.PlayerService
	matchSvc       *service.MatchService
	matchDetailSvc *service.MatchDetailService
}

func NewTrackerServer(playerSvc *service.PlayerService, matchSvc *service.MatchService, matchDetailSvc *service.MatchDetailService) *TrackerServer {
	return &TrackerServer{playerSvc: playerSvc, matchSvc: matchSvc, matchDetailSvc: matchDetailSvc}
}

func (s *TrackerServer) GetPlayer(ctx context.Context, req *connect.Request[valorantv1.PlayerRequest]) (*connect.Response[valorantv1.PlayerResponse], error) {
	fmt.Println("[BENCH] GetPlayer")
	start := time.Now()
	defer func() {
		fmt.Printf("[BENCH] GetPlayer END %d ms\n", time.Since(start).Milliseconds())
	}()

	player, err := s.playerSvc.GetPlayer(ctx, req.Msg.Name, req.Msg.Tag, req.Msg.Refresh)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	matches, err := s.matchSvc.GetMatchesFor(ctx, player.Puuid, req.Msg.Refresh)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	var totalKills, totalDeaths int
	for _, m := range matches {
		totalKills += m.PlayerStats.Kills
		totalDeaths += m.PlayerStats.Deaths
	}

	resp := &valorantv1.PlayerResponse{
		Puuid:        player.Puuid,
		Name:         player.Name,
		Tag:          player.Tag,
		Region:       player.Region,
		AccountLevel: int32(player.AccountLevel),
		Card:         player.Card,
		Title:        player.Title,
		CurrentTier: &valorantv1.Tier{
			Id:   int32(player.CurrentTier),
			Name: player.CurrentTierName,
		},
		CurrentRr:    int32(player.CurrentRR),
		TotalMatches: int32(len(matches)),
		KdRatio:      s.calculateKD(totalKills, totalDeaths),
		WinRate:      s.calculateWinRate(matches),
	}

	return connect.NewResponse(resp), nil
}

func (s *TrackerServer) GetMatches(ctx context.Context, req *connect.Request[valorantv1.MatchesRequest]) (*connect.Response[valorantv1.MatchesResponse], error) {
	fmt.Println("[BENCH] GetMatches")
	start := time.Now()
	defer func() {
		fmt.Printf("[BENCH] GetMatches END %d ms\n", time.Since(start).Milliseconds())
	}()

	matches, err := s.matchSvc.GetMatchesFor(ctx, req.Msg.Puuid, req.Msg.Refresh)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	var respMatches []*valorantv1.Match
	for _, m := range matches {
		rankingInTier := int32(0)
		mmrChange := int32(0)
		if m.MMRData != nil {
			rankingInTier = int32(m.MMRData.RankingInTier)
			mmrChange = int32(m.MMRData.MMRChange)
		}

		respMatches = append(respMatches, &valorantv1.Match{
			MatchId:   m.Match.MatchID,
			MapName:   m.Match.MapName,
			Mode:      m.Match.Mode,
			StartedAt: m.Match.StartedAt.Format(time.RFC3339),
			Tier: &valorantv1.Tier{
				Id:   int32(m.PlayerStats.Tier),
				Name: m.PlayerStats.TierName,
			},
			RankingInTier: rankingInTier,
			MmrChange:     mmrChange,
			Kills:         int32(m.PlayerStats.Kills),
			Deaths:        int32(m.PlayerStats.Deaths),
			Assists:       int32(m.PlayerStats.Assists),
			Score:         int32(m.PlayerStats.Score),
			Team:          m.PlayerStats.Team,
			HasWon:        m.PlayerStats.HasWon,
			Source:        m.Match.Source,
			TeamRedScore:  int32(m.Match.TeamRedScore),
			TeamBlueScore: int32(m.Match.TeamBlueScore),
			Cluster:       m.Match.Cluster,
			Version:       m.Match.Version,
			MapId:         m.Match.MapID,
			CharacterId:   m.PlayerStats.CharacterID,
			DamageTaken:   int32(m.PlayerStats.DamageTaken),
			DamageDealt:   int32(m.PlayerStats.DamageDealt),
		})
	}

	return connect.NewResponse(&valorantv1.MatchesResponse{Matches: respMatches}), nil
}

func (s *TrackerServer) calculateKD(kills, deaths int) float32 {
	if deaths == 0 {
		return float32(kills)
	}
	return float32(kills) / float32(deaths)
}

func (s *TrackerServer) calculateWinRate(matches []repository.MatchWithPlayers) float32 {
	if len(matches) == 0 {
		return 0
	}
	wins := 0
	for _, m := range matches {
		if m.PlayerStats.HasWon {
			wins++
		}
	}
	return float32(wins) / float32(len(matches))
}

func (s *TrackerServer) SearchSuggestions(ctx context.Context, req *connect.Request[valorantv1.SearchSuggestionsRequest]) (*connect.Response[valorantv1.SearchSuggestionsResponse], error) {
	suggestions, err := s.playerSvc.SearchSuggestions(ctx, req.Msg.Query)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&valorantv1.SearchSuggestionsResponse{
		Suggestions: suggestions,
	}), nil
}

func (s *TrackerServer) GetMatch(ctx context.Context, req *connect.Request[valorantv1.GetMatchRequest]) (*connect.Response[valorantv1.GetMatchResponse], error) {
	resp, err := s.matchDetailSvc.GetMatch(ctx, req.Msg.MatchId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(resp), nil
}

func (s *TrackerServer) GetPlayerByPuuid(ctx context.Context, req *connect.Request[valorantv1.GetPlayerByPuuidRequest]) (*connect.Response[valorantv1.PlayerResponse], error) {
	player, err := s.playerSvc.GetPlayerByPuuid(ctx, req.Msg.Puuid, req.Msg.Refresh)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	return connect.NewResponse(s.toProtoPlayer(player)), nil
}

func (s *TrackerServer) toProtoPlayer(p *domain.Player) *valorantv1.PlayerResponse {
	return &valorantv1.PlayerResponse{
		Puuid:        p.Puuid,
		Name:         p.Name,
		Tag:          p.Tag,
		Region:       p.Region,
		AccountLevel: int32(p.AccountLevel),
		Card:         p.Card,
		Title:        p.Title,
		CurrentTier:  &valorantv1.Tier{Id: int32(p.CurrentTier), Name: p.CurrentTierName},
		CurrentRr:    int32(p.CurrentRR),
	}
}
