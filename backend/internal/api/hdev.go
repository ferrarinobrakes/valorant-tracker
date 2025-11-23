package api

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"sync"
	"time"
	"valorant-tracker/internal/config"

	"github.com/valyala/fasthttp"
)

type HDevClient struct {
	apiKey      string
	client      *fasthttp.Client
	rateLimitMu sync.RWMutex
	rateLimit   RateLimitInfo
}

type RateLimitInfo struct {
	Bucket    string `json:"bucket"`
	Limit     int    `json:"limit"`
	Remaining int    `json:"remaining"`

	// seconds until reset
	Reset int `json:"reset"`

	UpdatedAt time.Time `json:"updated_at"`
}

func NewHDevClient(cfg *config.Config) *HDevClient {
	return &HDevClient{
		apiKey: cfg.HDevAPIKey,
		client: &fasthttp.Client{
			MaxConnsPerHost:     100,
			ReadTimeout:         10 * time.Second,
			WriteTimeout:        10 * time.Second,
			MaxIdleConnDuration: 1 * time.Minute,
		},
		rateLimit: RateLimitInfo{
			Limit:     90,
			Remaining: 90,
			Reset:     60,
			UpdatedAt: time.Now(),
		},
	}
}

func (c *HDevClient) GetRateLimitInfo() RateLimitInfo {
	c.rateLimitMu.RLock()
	defer c.rateLimitMu.RUnlock()
	return c.rateLimit
}

func (c *HDevClient) updateRateLimit(resp *fasthttp.Response) {
	c.rateLimitMu.Lock()
	defer c.rateLimitMu.Unlock()

	if bucket := string(resp.Header.Peek("X-Ratelimit-Bucket")); bucket != "" {
		c.rateLimit.Bucket = bucket
	}
	if limit := string(resp.Header.Peek("X-Ratelimit-Limit")); limit != "" {
		if val, err := strconv.Atoi(limit); err == nil {
			c.rateLimit.Limit = val
		}
	}
	if remaining := string(resp.Header.Peek("X-Ratelimit-Remaining")); remaining != "" {
		if val, err := strconv.Atoi(remaining); err == nil {
			c.rateLimit.Remaining = val
		}
	}
	if reset := string(resp.Header.Peek("X-Ratelimit-Reset")); reset != "" {
		if val, err := strconv.Atoi(reset); err == nil {
			c.rateLimit.Reset = val
		}
	}
	c.rateLimit.UpdatedAt = time.Now()
}

func (c *HDevClient) GetAccount(ctx context.Context, name, tag string) (*AccountResponse, error) {
	url := fmt.Sprintf("https://api.henrikdev.xyz/valorant/v2/account/%s/%s", name, tag)
	return doRequest[AccountResponse](ctx, c, url)
}

func (c *HDevClient) GetStoredMatches(ctx context.Context, region, puuid string) (*StoredMatchesResponse, error) {
	url := fmt.Sprintf("https://api.henrikdev.xyz/valorant/v1/by-puuid/stored-matches/%s/%s?mode=competitive", region, puuid)
	return doRequest[StoredMatchesResponse](ctx, c, url)
}

func (c *HDevClient) GetStoredMMRHistory(ctx context.Context, region, puuid string) (*StoredMMRHistoryResponse, error) {
	url := fmt.Sprintf("https://api.henrikdev.xyz/valorant/v1/by-puuid/stored-mmr-history/%s/%s", region, puuid)
	return doRequest[StoredMMRHistoryResponse](ctx, c, url)
}

func (c *HDevClient) GetV4Matches(ctx context.Context, region, puuid string) (*V4MatchesResponse, error) {
	url := fmt.Sprintf("https://api.henrikdev.xyz/valorant/v4/by-puuid/matches/%s/pc/%s", region, puuid)
	return doRequest[V4MatchesResponse](ctx, c, url)
}

func (c *HDevClient) GetMMRHistory(ctx context.Context, region, puuid string) (*MMRHistoryResponse, error) {
	url := fmt.Sprintf("https://api.henrikdev.xyz/valorant/v1/by-puuid/mmr-history/%s/%s", region, puuid)
	return doRequest[MMRHistoryResponse](ctx, c, url)
}

func (c *HDevClient) GetMMR(ctx context.Context, region, puuid string) (*MMRResponse, error) {
	url := fmt.Sprintf("https://api.henrikdev.xyz/valorant/v3/by-puuid/mmr/%s/pc/%s", region, puuid)
	return doRequest[MMRResponse](ctx, c, url)
}

func doRequest[T any](ctx context.Context, client *HDevClient, url string) (*T, error) {
	req := fasthttp.AcquireRequest()
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(resp)

	req.SetRequestURI(url)
	req.Header.SetMethod(fasthttp.MethodGet)
	req.Header.Set("Authorization", client.apiKey)

	deadline, ok := ctx.Deadline()
	if ok {
		if err := client.client.DoDeadline(req, resp, deadline); err != nil {
			return nil, err
		}
	} else {
		if err := client.client.Do(req, resp); err != nil {
			return nil, err
		}
	}

	client.updateRateLimit(resp)

	if resp.StatusCode() != fasthttp.StatusOK {
		return nil, fmt.Errorf("API error: %d", resp.StatusCode())
	}

	var result T
	if err := json.Unmarshal(resp.Body(), &result); err != nil {
		return nil, err
	}
	return &result, nil
}

type AccountResponse struct {
	Status int         `json:"status"`
	Data   AccountData `json:"data"`
}

type AccountData struct {
	Puuid        string   `json:"puuid"`
	Region       string   `json:"region"`
	AccountLevel int      `json:"account_level"`
	Name         string   `json:"name"`
	Tag          string   `json:"tag"`
	Card         string   `json:"card"`
	Title        string   `json:"title"`
	Platforms    []string `json:"platforms"`
	UpdatedAt    string   `json:"updated_at"`
}

type StoredMatchesResponse struct {
	Status  int           `json:"status"`
	Results ResponseStats `json:"results"`
	Data    []StoredMatch `json:"data"`
}

type StoredMatch struct {
	Meta  StoredMatchMeta  `json:"meta"`
	Stats StoredMatchStats `json:"stats"`
	Teams StoredMatchTeams `json:"teams"`
}

type StoredMatchMeta struct {
	ID  string `json:"id"`
	Map struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"map"`
	StartedAt time.Time `json:"started_at"`
	Season    struct {
		ID    string `json:"id"`
		Short string `json:"short"`
	} `json:"season"`
	Region  string `json:"region"`
	Cluster string `json:"cluster"`
	Version string `json:"version"`
}

type StoredMatchStats struct {
	Tier      int    `json:"tier"`
	Kills     int    `json:"kills"`
	Deaths    int    `json:"deaths"`
	Assists   int    `json:"assists"`
	Score     int    `json:"score"`
	Team      string `json:"team"`
	Character struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"character"`
	Damage struct {
		Made     int `json:"made"`
		Received int `json:"received"`
	} `json:"damage"`
}

type StoredMatchTeams struct {
	Red  int `json:"red"`
	Blue int `json:"blue"`
}

type StoredMMRHistoryResponse struct {
	Status  int                    `json:"status"`
	Results ResponseStats          `json:"results"`
	Data    []StoredMMRHistoryItem `json:"data"`
}

type StoredMMRHistoryItem struct {
	MatchID string `json:"match_id"`
	Tier    struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	} `json:"tier"`
	RankingInTier int       `json:"ranking_in_tier"`
	LastMmrChange int       `json:"last_mmr_change"`
	Elo           int       `json:"elo"`
	Date          time.Time `json:"date"`
}

type ResponseStats struct {
	Total    int `json:"total"`
	Returned int `json:"returned"`
}

type V4MatchesResponse struct {
	Status int           `json:"status"`
	Data   []V4MatchData `json:"data"`
}

type V4MatchData struct {
	Metadata V4MatchMetadata `json:"metadata"`
	Players  []V4Player      `json:"players"`
	Teams    []V4Team        `json:"teams"`
}

type V4MatchMetadata struct {
	MatchID string `json:"match_id"`
	Region  string `json:"region"`
	Cluster string `json:"cluster"`
	Map     struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"map"`
	StartedAt time.Time `json:"started_at"`
	Season    struct {
		ID    string `json:"id"`
		Short string `json:"short"`
	} `json:"season"`
	GameVersion string `json:"game_version"`
}

type V4Player struct {
	Puuid string `json:"puuid"`
	Name  string `json:"name"`
	Tag   string `json:"tag"`
	Agent struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"agent"`
	Stats struct {
		Score   int `json:"score"`
		Kills   int `json:"kills"`
		Deaths  int `json:"deaths"`
		Assists int `json:"assists"`
		Damage  struct {
			Made     int `json:"dealt"`
			Received int `json:"received"`
		} `json:"damage"`
	} `json:"stats"`
	Tier struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	} `json:"tier"`
	AccountLevel  int `json:"account_level"`
	Customization struct {
		Card  string `json:"card"`
		Title string `json:"title"`
	} `json:"customization"`
	TeamID   string `json:"team_id"`
	Behavior struct {
		AfkRounds    float64 `json:"afk_rounds"`
		FriendlyFire struct {
			Incoming float64 `json:"incoming"`
			Outgoing float64 `json:"outgoing"`
		} `json:"friendly_fire"`
		RoundsInSpawn float64 `json:"rounds_in_spawn"`
	} `json:"behavior"`
}

type V4Team struct {
	TeamID string `json:"team_id"`
	Won    bool   `json:"won"`
	Rounds struct {
		Won int `json:"won"`
	} `json:"rounds"`
}

type MMRHistoryResponse struct {
	Status int              `json:"status"`
	Data   []MMRHistoryItem `json:"data"`
}

type MMRHistoryItem struct {
	CurrentTier         int    `json:"currenttier"`
	CurrentTierPatched  string `json:"currenttierpatched"`
	MatchID             string `json:"match_id"`
	RankingInTier       int    `json:"ranking_in_tier"`
	MmrChangeToLastGame int    `json:"mmr_change_to_last_game"`
	Elo                 int    `json:"elo"`
	Date                string `json:"date"`
}

type MMRResponse struct {
	Status int        `json:"status"`
	Data   MMRCurrent `json:"data"`
}

type MMRCurrent struct {
	Current struct {
		Tier struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
		} `json:"tier"`
		RR int `json:"rr"`
	} `json:"current"`
}

func (c *HDevClient) GetMatchV2(ctx context.Context, matchID string) (*MatchV2Response, error) {
	url := fmt.Sprintf("https://api.henrikdev.xyz/valorant/v2/match/%s", matchID)
	return doRequest[MatchV2Response](ctx, c, url)
}

type MatchV2Response struct {
	Status int `json:"status"`
	Data   struct {
		Metadata struct {
			Map          string `json:"map"`
			GameVersion  string `json:"game_version"`
			Region       string `json:"region"`
			Cluster      string `json:"cluster"`
			Mode         string `json:"mode"`
			SeasonID     string `json:"season_id"`
			Matchid      string `json:"matchid"`
			RoundsPlayed int    `json:"rounds_played"`
			GameStart    int64  `json:"game_start"`
		} `json:"metadata"`
		Players struct {
			AllPlayers []struct {
				Puuid              string `json:"puuid"`
				Name               string `json:"name"`
				Tag                string `json:"tag"`
				Team               string `json:"team"`
				Level              int    `json:"level"`
				Character          string `json:"character"`
				Currenttier        int    `json:"currenttier"`
				CurrenttierPatched string `json:"currenttier_patched"`
				PlayerCard         string `json:"player_card"`
				PlayerTitle        string `json:"player_title"`
				Stats              struct {
					Score   int `json:"score"`
					Kills   int `json:"kills"`
					Deaths  int `json:"deaths"`
					Assists int `json:"assists"`
				} `json:"stats"`
				DamageMade     int `json:"damage_made"`
				DamageReceived int `json:"damage_received"`
			} `json:"all_players"`
		} `json:"players"`
		Teams struct {
			Red struct {
				RoundsWon int `json:"rounds_won"`
			} `json:"red"`
			Blue struct {
				RoundsWon int `json:"rounds_won"`
			} `json:"blue"`
		} `json:"teams"`
	} `json:"data"`
}
