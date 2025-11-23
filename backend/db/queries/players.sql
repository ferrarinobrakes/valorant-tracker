-- name: GetPlayerByPuuid :one
SELECT * FROM players
WHERE puuid = ? LIMIT 1;

-- name: UpsertPlayer :exec
INSERT INTO players (
    puuid, name, tag, region, account_level, card, title,
    current_tier, current_tier_name, current_rr,
    is_partial_fetch, last_fetch_at, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(puuid) DO UPDATE SET
    name = excluded.name,
    tag = excluded.tag,
    region = excluded.region,
    account_level = excluded.account_level,
    card = excluded.card,
    title = excluded.title,
    current_tier = excluded.current_tier,
    current_tier_name = excluded.current_tier_name,
    current_rr = excluded.current_rr,
    is_partial_fetch = excluded.is_partial_fetch,
    last_fetch_at = excluded.last_fetch_at,
    updated_at = excluded.updated_at;

-- name: GetPlayerLastFetchAt :one
SELECT last_fetch_at, is_partial_fetch FROM players
WHERE puuid = ? LIMIT 1;

-- name: UpdatePlayerLastFetchAt :exec
UPDATE players
SET last_fetch_at = ?, updated_at = ?
WHERE puuid = ?;

-- name: SearchPlayers :many
SELECT * FROM players
WHERE name LIKE ? OR tag LIKE ?
ORDER BY account_level DESC
LIMIT ?;

-- name: GetPlayerByNameTag :one
SELECT * FROM players
WHERE name = ? AND tag = ?
LIMIT 1;

-- name: UpdatePlayerPartialFetch :exec
UPDATE players
SET is_partial_fetch = ?, updated_at = ?
WHERE puuid = ?;
