-- +goose Up
CREATE TABLE IF NOT EXISTS tg_message (
    id uuid PRIMARY KEY,
    message_id bigint NOT NULL,
    chat_id bigint NOT NULL,
    thread_id bigint NOT NULL,
    user_id bigint NOT NULL,
    type text NOT NULL CHECK (type IN ('from_user', 'to_user')),
    date timestamptz NOT NULL,
    text text NOT NULL DEFAULT '',
    UNIQUE (chat_id, message_id)
);

-- +goose Down
DROP TABLE IF EXISTS tg_message;
