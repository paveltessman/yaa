-- +goose NO TRANSACTION

-- +goose Up
CREATE INDEX CONCURRENTLY IF NOT EXISTS session_history_parent_idx
    ON session_history (parent_session_id);

CREATE INDEX CONCURRENTLY IF NOT EXISTS session_history_pipeline_idx
    ON session_history (pipeline, date);

-- +goose Down
DROP INDEX CONCURRENTLY IF EXISTS session_history_parent_idx;

DROP INDEX CONCURRENTLY IF EXISTS session_history_pipeline_idx;
