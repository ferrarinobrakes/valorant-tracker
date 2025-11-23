package constants

import "time"

const (
	PlayerCacheTTL   = 5 * time.Minute
	MatchCacheTTL    = 10 * time.Minute
	SearchCacheTTL   = 2 * time.Minute
	PlayerRefreshTTL = 5 * time.Minute
	MatchRefreshTTL  = 5 * time.Minute
	LastFetchDelay   = 10 * time.Second
)

const (
	ExternalAPITimeout = 10 * time.Second
	DatabaseTimeout    = 5 * time.Second
	RequestTimeout     = 30 * time.Second
)

const (
	DBMaxOpenConns    = 100
	DBMaxIdleConns    = 10
	DBConnMaxLifetime = 1 * time.Hour
	DBMaxIdleTime     = 10 * time.Minute
	DBBatchSize       = 100
)

const (
	ShutdownTimeout = 5 * time.Second
)

const (
	SearchSuggestionLimit = 10
)
