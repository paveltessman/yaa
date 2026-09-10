-- +goose Up
CREATE TABLE IF NOT EXISTS session_history (
    session_id uuid PRIMARY KEY,
    parent_session_id uuid,
    pipeline text NOT NULL,
    date timestamptz NOT NULL,
    entries jsonb NOT NULL DEFAULT '[]'::jsonb
);

-- +goose Down
DROP TABLE IF EXISTS session_history;
