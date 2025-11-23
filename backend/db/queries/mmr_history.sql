-- name: UpsertMMRHistory :exec
INSERT INTO mmr_histories (
    id, match_id, puuid, tier, tier_name, ranking_in_tier,
    mmr_change, elo, date, source, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(match_id, puuid) DO UPDATE SET
    tier = excluded.tier,
    tier_name = excluded.tier_name,
    ranking_in_tier = excluded.ranking_in_tier,
    mmr_change = excluded.mmr_change,
    elo = excluded.elo,
    date = excluded.date,
    source = excluded.source,
    updated_at = excluded.updated_at;

-- name: GetMMRHistoryByPuuid :many
SELECT * FROM mmr_histories
WHERE puuid = ?
ORDER BY date DESC
LIMIT ?;
