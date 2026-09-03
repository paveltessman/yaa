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
