package tgupdates

import (
	"context"
	"fmt"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/paveltessman/yaa/pipelines/telegram/updates/models"
	"github.com/paveltessman/yaa/platform/db/internal/sqlc"
)

var _ models.DBRepo = (*Repo)(nil)

type Repo struct {
	queries *sqlc.Queries
}

func New(pool *pgxpool.Pool) *Repo {
	if pool == nil {
		panic("pool object is nil")
	}

	repo := Repo{queries: sqlc.New(pool)}
	return &repo
}

func (r *Repo) StoreMessage(ctx context.Context, message *models.Message) error {
	params := sqlc.StoreTgMessageParams{
		ID:        uuid.NewV7(),
		MessageID: message.ID,
		ChatID:    message.ChatID,
		ThreadID:  message.ThreadID,
		UserID:    message.UserID,
		Type:      string(message.Type),
		Date:      message.Date,
		Text:      message.Text,
	}

	if err := r.queries.StoreTgMessage(ctx, params); err != nil {
		return fmt.Errorf("tgupdates: can't store message %d of chat %d: %w", message.ID, message.ChatID, err)
	}
	return nil
}

func (r *Repo) LoadThread(ctx context.Context, chatID, threadID int64) ([]*models.Message, error) {
	params := sqlc.LoadTgThreadParams{
		ChatID:   chatID,
		ThreadID: threadID,
	}

	rows, err := r.queries.LoadTgThread(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("tgupdates: can't load thread %d of chat %d: %w", threadID, chatID, err)
	}

	messages := make([]*models.Message, 0, len(rows))
	for _, row := range rows {
		messages = append(messages, toMessage(row))
	}
	return messages, nil
}

func toMessage(row sqlc.TgMessage) *models.Message {
	message := models.Message{
		ID:       row.MessageID,
		ChatID:   row.ChatID,
		ThreadID: row.ThreadID,
		UserID:   row.UserID,
		Type:     models.MessageType(row.Type),
		Date:     row.Date,
		Text:     row.Text,
	}
	return &message
}
