package service

import (
	"context"
	"time"
	valorantv1 "valorant-tracker/gen/proto/valorant/v1"
	"valorant-tracker/internal/api"
	"valorant-tracker/internal/domain"
	"valorant-tracker/internal/repository"

	"github.com/rs/zerolog"
)

type MatchDetailService struct {
	hdev       *api.HDevClient
	matchRepo  *repository.MatchRepository
	playerRepo *repository.PlayerRepository
	logger     zerolog.Logger
}

func NewMatchDetailService(hdev *api.HDevClient, matchRepo *repository.MatchRepository, playerRepo *repository.PlayerRepository, logger zerolog.Logger) *MatchDetailService {
	return &MatchDetailService{hdev: hdev, matchRepo: matchRepo, playerRepo: playerRepo, logger: logger}
}

func (s *MatchDetailService) GetMatch(ctx context.Context, matchID string) (*valorantv1.GetMatchResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*30)
	defer cancel()

	s.logger.Debug().Str("match_id", matchID).Msg("getting match")

	matches, err := s.matchRepo.GetByMatchID(ctx, matchID)
	if err != nil || len(matches) == 0 {
		s.logger.Debug().Str("match_id", matchID).Msg("match not found in cache, fetching from API")
		return s.fetchAndStoreMatch(ctx, matchID)
	}

	if len(matches) != 10 {
		s.logger.Warn().Str("match_id", matchID).Int("player_count", len(matches)).Msg("incomplete match data, refetching")
		_, err := s.fetchAndStoreMatch(ctx, matchID)
		if err != nil {
			s.logger.Error().Err(err).Str("match_id", matchID).Msg("failed to fetch and store match")
			return nil, err
		}
		return s.GetMatch(ctx, matchID)
	}

	metadata, _ := s.matchRepo.GetMatchMetadata(ctx, matchID)
	s.logger.Info().Str("match_id", matchID).Msg("match found in cache")
	return s.buildResponse(metadata, matches), nil
}

var mapNameToID = map[string]string{
	"Bind":           "2c9d57ec-4431-9c5e-2939-8f9ef6dd5cba",
	"Ascent":         "7eaecc1b-4337-bbf6-6ab9-04b8f06b3319",
	"Fracture":       "b529448b-4d60-346e-e89e-00a4c527a405",
	"Breeze":         "2fb9a4fd-47b8-4e7d-a969-74b4046ebd53",
	"District":       "690b3ed0-4dff-945b-8223-6da834e30d24",
	"Kasbah":         "8edabed9-466a-44c7-96ee-199b73104b00",
	"Drift":          "56801fc8-4d09-1818-a989-49bf2e17bb5f",
	"Piazza":         "de28aa9b-4cbe-1003-320e-6cb3ec309557",
	"Lotus":          "2fe4ed3a-450a-948b-6d6b-e89a78e680a9",
	"Split":          "d960549e-485c-e861-8d71-aa9d1aed12a2",
	"Abyss":          "224b0a95-48b9-f703-1bd8-67aca101a61f",
	"Sunset":         "92584fbe-486a-b1b2-9faa-39b0f486b498",
	"Basic Training": "1f10dab3-4294-3827-fa35-c2aa00213cf3",
	"Pearl":          "fd267378-4d1d-484f-ff52-77821ed10dc2",
	"Icebox":         "e2ad5c54-4114-a870-9641-8ea21279579a",
	"The Range":      "ee613ee9-28b7-4beb-9666-08db13bb2244",
	"Corrode":        "1c18ab1f-420d-0d8b-71d0-77ad3c439115",
	"Haven":          "2bee0dc9-4ffe-519b-1cbd-7fbe763a6047",
}

var characterNameToID = map[string]string{
	"Gekko":     "e370fa57-4757-3604-3648-499e1f642d3f",
	"Fade":      "dade69b4-4f5a-8528-247b-219e5a1facd6",
	"Breach":    "5f8d3a7f-467b-97f3-062c-13acf203c006",
	"Deadlock":  "cc8b64c8-4b25-4ff9-6e7f-37b4da43d235",
	"Tejo":      "b444168c-4e35-8076-db47-ef9bf368f384",
	"Raze":      "f94c3b30-42be-e959-889c-5aa313dba261",
	"Chamber":   "22697a3d-45bf-8dd7-4fec-84a9e28c69d7",
	"KAY/O":     "601dbbe7-43ce-be57-2a40-4abd24953621",
	"Skye":      "6f2a04ca-43e0-be17-7f36-b3908627744d",
	"Cypher":    "117ed9e3-49f3-6512-3ccf-0cada7e3823b",
	"Sova":      "320b2a48-4d9b-a075-30f1-1f93a9b638fa",
	"Killjoy":   "1e58de9c-4950-5125-93e9-a0aee9f98746",
	"Harbor":    "95b78ed7-4637-86d9-7e41-71ba8c293152",
	"Vyse":      "efba5359-4016-a1e5-7626-b1ae76895940",
	"Viper":     "707eab51-4836-f488-046a-cda6bf494859",
	"Phoenix":   "eb93336a-449b-9c1b-0a54-a891f7921d69",
	"Veto":      "92eeef5d-43b5-1d4a-8d03-b3927a09034b",
	"Astra":     "41fb69c1-4189-7b37-f117-bcaf1e96f1bf",
	"Brimstone": "9f0d8ba9-4140-b941-57d3-a7ad57c6b417",
	"Iso":       "0e38b510-41a8-5780-5e8f-568b2a4f2d6c",
	"Clove":     "1dbf2edd-4729-0984-3115-daa5eed44993",
	"Neon":      "bb2a4828-46eb-8cd1-e765-15848195d751",
	"Yoru":      "7f94d92c-4234-0a36-9646-3a87eb8b5c89",
	"Waylay":    "df1cb487-4902-002e-5c17-d28e83e78588",
	"Sage":      "569fdd95-4d10-43ab-ca70-79becc718b46",
	"Reyna":     "a3bfb853-43b2-7238-a4f1-ad90e9e46bcc",
	"Omen":      "8e253930-4c05-31dd-1b6c-968525494517",
	"Jett":      "add6443a-41bd-e414-f6ad-e58d267f4e95",
}

func (s *MatchDetailService) fetchAndStoreMatch(ctx context.Context, matchID string) (*valorantv1.GetMatchResponse, error) {
	resp, err := s.hdev.GetMatchV2(ctx, matchID)
	if err != nil {
		return nil, err
	}

	var players []domain.Player
	var matchPlayers []domain.MatchPlayer

	match := domain.Match{
		MatchID:       resp.Data.Metadata.Matchid,
		MapName:       resp.Data.Metadata.Map,
		MapID:         mapNameToID[resp.Data.Metadata.Map],
		Mode:          resp.Data.Metadata.Mode,
		StartedAt:     time.Unix(int64(resp.Data.Metadata.GameStart), 0),
		SeasonID:      resp.Data.Metadata.SeasonID,
		TeamRedScore:  resp.Data.Teams.Red.RoundsWon,
		TeamBlueScore: resp.Data.Teams.Blue.RoundsWon,
		Region:        resp.Data.Metadata.Region,
		Cluster:       resp.Data.Metadata.Cluster,
		Version:       resp.Data.Metadata.GameVersion,
		Source:        "v2",
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	for _, p := range resp.Data.Players.AllPlayers {
		players = append(players, domain.Player{
			Puuid:           p.Puuid,
			Name:            p.Name,
			Tag:             p.Tag,
			Region:          resp.Data.Metadata.Region,
			AccountLevel:    p.Level,
			Card:            p.PlayerCard,
			Title:           p.PlayerTitle,
			CurrentTier:     p.Currenttier,
			CurrentTierName: p.CurrenttierPatched,
			LastFetchAt:     time.Now(),
		})

		matchPlayers = append(matchPlayers, domain.MatchPlayer{
			MatchID:     resp.Data.Metadata.Matchid,
			Puuid:       p.Puuid,
			Name:        p.Name,
			Tag:         p.Tag,
			Tier:        p.Currenttier,
			TierName:    p.CurrenttierPatched,
			Kills:       p.Stats.Kills,
			Deaths:      p.Stats.Deaths,
			Assists:     p.Stats.Assists,
			Score:       p.Stats.Score,
			Team:        p.Team,
			HasWon:      (p.Team == "Red" && resp.Data.Teams.Red.RoundsWon > resp.Data.Teams.Blue.RoundsWon) || (p.Team == "Blue" && resp.Data.Teams.Blue.RoundsWon > resp.Data.Teams.Red.RoundsWon),
			CharacterID: characterNameToID[p.Character],
			DamageTaken: p.DamageReceived,
			DamageDealt: p.DamageMade,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		})
	}

	for _, p := range players {
		s.playerRepo.Upsert(ctx, &p)
	}

	s.matchRepo.UpsertMatch(ctx, &match)
	for _, mp := range matchPlayers {
		s.matchRepo.UpsertMatchPlayer(ctx, &mp)
	}

	metadata, _ := s.matchRepo.GetMatchMetadata(ctx, matchID)
	storedPlayers, _ := s.matchRepo.GetByMatchID(ctx, matchID)

	return s.buildResponse(metadata, storedPlayers), nil
}

func (s *MatchDetailService) buildResponse(metadata *domain.Match, players []domain.MatchPlayer) *valorantv1.GetMatchResponse {
	if metadata == nil {
		return &valorantv1.GetMatchResponse{}
	}

	return &valorantv1.GetMatchResponse{
		Metadata: &valorantv1.MatchMetadata{
			MatchId:       metadata.MatchID,
			MapName:       metadata.MapName,
			MapId:         metadata.MapID,
			GameVersion:   metadata.Version,
			TeamRedScore:  int32(metadata.TeamRedScore),
			TeamBlueScore: int32(metadata.TeamBlueScore),
			Region:        metadata.Region,
			Cluster:       metadata.Cluster,
			Mode:          metadata.Mode,
			SeasonId:      metadata.SeasonID,
			GameStart:     metadata.StartedAt.Unix(),
			RoundsPlayed:  int32(metadata.TeamRedScore + metadata.TeamBlueScore),
		},
		Players: s.toProtoPlayers(players),
	}
}

func (s *MatchDetailService) toProtoPlayers(players []domain.MatchPlayer) []*valorantv1.PlayerMatch {
	var protoPlayers []*valorantv1.PlayerMatch
	for _, p := range players {
		protoPlayers = append(protoPlayers, &valorantv1.PlayerMatch{
			Puuid:       p.Puuid,
			Name:        p.Name,
			Tag:         p.Tag,
			Team:        p.Team,
			Agent:       p.CharacterID,
			CharacterId: p.CharacterID,
			Kills:       int32(p.Kills),
			Deaths:      int32(p.Deaths),
			Assists:     int32(p.Assists),
			Score:       int32(p.Score),
			DamageTaken: int32(p.DamageTaken),
			DamageDealt: int32(p.DamageDealt),
			HasWon:      p.HasWon,
			Tier: &valorantv1.Tier{
				Id:   int32(p.Tier),
				Name: p.TierName,
			},
		})
	}
	return protoPlayers
}
