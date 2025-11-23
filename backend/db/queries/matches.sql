-- name: GetMatchesByPuuid :many
SELECT m.* FROM matches m
INNER JOIN match_players mp ON m.match_id = mp.match_id
WHERE mp.puuid = ?
ORDER BY m.started_at DESC;

-- name: GetMatchPlayersByMatchIDs :many
SELECT * FROM match_players
WHERE puuid = ? AND match_id IN (sqlc.slice('match_ids'));

-- name: GetMMRHistoryByMatchIDs :many
SELECT * FROM mmr_histories
WHERE puuid = ? AND match_id IN (sqlc.slice('match_ids'));

-- name: UpsertMatch :exec
INSERT INTO matches (
    match_id, map_name, map_id, mode, started_at, season_id,
    team_red_score, team_blue_score, region, cluster, version,
    source, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(match_id) DO UPDATE SET
    map_name = excluded.map_name,
    map_id = excluded.map_id,
    mode = excluded.mode,
    started_at = excluded.started_at,
    season_id = excluded.season_id,
    team_red_score = excluded.team_red_score,
    team_blue_score = excluded.team_blue_score,
    region = excluded.region,
    cluster = excluded.cluster,
    version = excluded.version,
    source = excluded.source,
    updated_at = excluded.updated_at;

-- name: UpsertMatchPlayer :exec
INSERT INTO match_players (
    match_id, puuid, name, tag, tier, tier_name,
    kills, deaths, assists, score, team, has_won,
    character_id, damage_taken, damage_dealt,
    created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(match_id, puuid) DO UPDATE SET
    name = excluded.name,
    tag = excluded.tag,
    tier = excluded.tier,
    tier_name = excluded.tier_name,
    kills = excluded.kills,
    deaths = excluded.deaths,
    assists = excluded.assists,
    score = excluded.score,
    team = excluded.team,
    has_won = excluded.has_won,
    character_id = excluded.character_id,
    damage_taken = excluded.damage_taken,
    damage_dealt = excluded.damage_dealt,
    updated_at = excluded.updated_at;

-- name: GetLatestMatchDate :one
SELECT m.started_at FROM matches m
INNER JOIN match_players mp ON m.match_id = mp.match_id
WHERE mp.puuid = ?
ORDER BY m.started_at DESC
LIMIT 1;

-- name: CountStoredGames :one
SELECT COUNT(*) as count FROM matches m
INNER JOIN match_players mp ON m.match_id = mp.match_id
WHERE mp.puuid = ? AND m.source = 'stored';

-- name: GetMatchPlayersByMatchID :many
SELECT * FROM match_players
WHERE match_id = ?;

-- name: GetMatchMetadata :one
SELECT * FROM matches
WHERE match_id = ?
LIMIT 1;
