-- name: StoreTgMessage :exec
INSERT INTO tg_message (id, message_id, chat_id, thread_id, user_id, type, date, text)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
ON CONFLICT (chat_id, message_id) DO NOTHING;

-- name: LoadTgThread :many
SELECT id, message_id, chat_id, thread_id, user_id, type, date, text
FROM tg_message
WHERE chat_id = $1 AND thread_id = $2
ORDER BY date, message_id;
