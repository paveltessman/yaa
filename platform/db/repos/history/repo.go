package history

import (
	"context"
	"encoding/json"
	"fmt"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	ports "github.com/paveltessman/yaa/pipelines/shared/ports/history"
	"github.com/paveltessman/yaa/platform/db/internal/sqlc"
)

var _ ports.HistoryService = (*Repo)(nil)

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

func (r *Repo) Save(ctx context.Context, record *ports.Record) error {
	entries, err := toEntries(record.Entries)
	if err != nil {
		return fmt.Errorf("history: can't store session %s: %w", record.SessionID, err)
	}

	params := sqlc.StoreSessionHistoryParams{
		SessionID:       record.SessionID,
		ParentSessionID: toParent(record.ParentID),
		Pipeline:        record.Pipeline,
		Date:            record.Date,
		Entries:         entries,
	}

	if err := r.queries.StoreSessionHistory(ctx, params); err != nil {
		return fmt.Errorf("history: can't store session %s: %w", record.SessionID, err)
	}
	return nil
}

// toEntries writes the entries as a json list. A record without entries gives
// an empty list, because the column takes no null.
func toEntries(entries []ports.Entry) ([]byte, error) {
	if entries == nil {
		entries = []ports.Entry{}
	}
	return json.Marshal(entries)
}

func toParent(parentID uuid.UUID) *uuid.UUID {
	if parentID == uuid.Nil() {
		return nil
	}
	return &parentID
}
