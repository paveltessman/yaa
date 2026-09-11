-- name: StoreSessionHistory :exec
INSERT INTO session_history (session_id, parent_session_id, pipeline, date, entries)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (session_id) DO UPDATE
SET parent_session_id = EXCLUDED.parent_session_id,
    pipeline = EXCLUDED.pipeline,
    date = EXCLUDED.date,
    entries = EXCLUDED.entries;

-- name: ListSessionHistory :many
SELECT session_id, parent_session_id, pipeline, date, entries
FROM session_history
ORDER BY date DESC, session_id DESC
LIMIT $1;

-- name: GetSessionHistory :one
SELECT session_id, parent_session_id, pipeline, date, entries
FROM session_history
WHERE session_id = $1;
