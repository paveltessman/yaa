-- name: StoreTgMessage :exec
INSERT INTO tg_message (id, message_id, chat_id, thread_id, user_id, type, date, text)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
ON CONFLICT (chat_id, message_id) DO NOTHING;
