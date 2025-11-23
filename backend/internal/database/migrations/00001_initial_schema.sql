-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS players (
    puuid TEXT PRIMARY KEY NOT NULL,
    name TEXT NOT NULL,
    tag TEXT NOT NULL,
    region TEXT NOT NULL,
    account_level INTEGER NOT NULL DEFAULT 0,
    card TEXT NOT NULL DEFAULT '',
    title TEXT NOT NULL DEFAULT '',
    current_tier INTEGER NOT NULL DEFAULT 0,
    current_tier_name TEXT NOT NULL DEFAULT '',
    current_rr INTEGER NOT NULL DEFAULT 0,
    is_partial_fetch BOOLEAN NOT NULL DEFAULT FALSE,
    last_fetch_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_players_name_tag ON players(name, tag);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS matches (
    match_id TEXT PRIMARY KEY NOT NULL,
    map_name TEXT NOT NULL,
    map_id TEXT NOT NULL,
    mode TEXT NOT NULL,
    started_at DATETIME NOT NULL,
    season_id TEXT NOT NULL,
    team_red_score INTEGER NOT NULL DEFAULT 0,
    team_blue_score INTEGER NOT NULL DEFAULT 0,
    region TEXT NOT NULL,
    cluster TEXT NOT NULL,
    version TEXT NOT NULL,
    source TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_matches_started_at ON matches(started_at);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS match_players (
    match_id TEXT NOT NULL,
    puuid TEXT NOT NULL,
    name TEXT NOT NULL,
    tag TEXT NOT NULL DEFAULT '',
    tier INTEGER NOT NULL DEFAULT 0,
    tier_name TEXT NOT NULL DEFAULT '',
    kills INTEGER NOT NULL DEFAULT 0,
    deaths INTEGER NOT NULL DEFAULT 0,
    assists INTEGER NOT NULL DEFAULT 0,
    score INTEGER NOT NULL DEFAULT 0,
    team TEXT NOT NULL,
    has_won BOOLEAN NOT NULL DEFAULT FALSE,
    character_id TEXT NOT NULL,
    damage_taken INTEGER NOT NULL DEFAULT 0,
    damage_dealt INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (match_id, puuid),
    FOREIGN KEY (match_id) REFERENCES matches(match_id) ON DELETE CASCADE,
    FOREIGN KEY (puuid) REFERENCES players(puuid) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_match_players_puuid ON match_players(puuid);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS mmr_histories (
    id TEXT PRIMARY KEY NOT NULL,
    match_id TEXT NOT NULL,
    puuid TEXT NOT NULL,
    tier INTEGER NOT NULL DEFAULT 0,
    tier_name TEXT NOT NULL DEFAULT '',
    ranking_in_tier INTEGER NOT NULL DEFAULT 0,
    mmr_change INTEGER NOT NULL DEFAULT 0,
    elo INTEGER NOT NULL DEFAULT 0,
    date DATETIME NOT NULL,
    source TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (match_id) REFERENCES matches(match_id) ON DELETE CASCADE,
    FOREIGN KEY (puuid) REFERENCES players(puuid) ON DELETE CASCADE
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_mmr_match_player ON mmr_histories(match_id, puuid);
CREATE INDEX IF NOT EXISTS idx_mmr_history_player_date ON mmr_histories(puuid, date);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS mmr_histories;
DROP TABLE IF EXISTS match_players;
DROP TABLE IF EXISTS matches;
DROP TABLE IF EXISTS players;
-- +goose StatementEnd
