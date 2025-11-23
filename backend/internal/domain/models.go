package domain

import (
	"time"
)

type Player struct {
	Puuid           string
	Name            string
	Tag             string
	Region          string
	AccountLevel    int
	Card            string
	Title           string
	CurrentTier     int
	CurrentTierName string
	CurrentRR       int
	IsPartialFetch  bool
	LastFetchAt     time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type Match struct {
	MatchID       string
	MapName       string
	MapID         string
	Mode          string
	StartedAt     time.Time
	SeasonID      string
	TeamRedScore  int
	TeamBlueScore int
	Region        string
	Cluster       string
	Version       string
	Source        string // "stored", "v4", "v2"
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type MatchPlayer struct {
	MatchID     string
	Puuid       string
	Name        string
	Tier        int
	TierName    string
	Kills       int
	Deaths      int
	Assists     int
	Score       int
	Team        string // "Red" or "Blue"
	HasWon      bool
	CharacterID string
	DamageTaken int
	Tag         string
	DamageDealt int
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type MMRHistory struct {
	ID            string // nanoid
	MatchID       string
	Puuid         string
	Tier          int
	TierName      string
	RankingInTier int // RR (0-100)
	MMRChange     int // +20, -18, etc.
	Elo           int // Hidden MMR
	Date          time.Time
	Source        string // "stored-mmr-history", "mmr-history", "live"
	CreatedAt     time.Time
	UpdatedAt     time.Time
}
