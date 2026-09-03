-- +goose NO TRANSACTION

-- +goose Up
CREATE INDEX CONCURRENTLY IF NOT EXISTS tg_message_thread_idx
    ON tg_message (chat_id, thread_id, date, message_id);

-- +goose Down
DROP INDEX CONCURRENTLY IF EXISTS tg_message_thread_idx;
