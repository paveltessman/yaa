package history

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"uuid"

	"github.com/jackc/pgx/v5"
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

func (r *Repo) List(ctx context.Context, limit int32) ([]*ports.Record, error) {
	rows, err := r.queries.ListSessionHistory(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("history: can't list the sessions: %w", err)
	}

	records := make([]*ports.Record, 0, len(rows))
	for _, row := range rows {
		record, err := toRecord(row)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	return records, nil
}

func (r *Repo) Get(ctx context.Context, sessionID uuid.UUID) (*ports.Record, error) {
	row, err := r.queries.GetSessionHistory(ctx, sessionID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("history: session %s: %w", sessionID, ports.ErrNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("history: can't read session %s: %w", sessionID, err)
	}
	return toRecord(row)
}

func toRecord(row sqlc.SessionHistory) (*ports.Record, error) {
	entries, err := fromEntries(row.Entries)
	if err != nil {
		return nil, fmt.Errorf("history: can't read session %s: %w", row.SessionID, err)
	}

	record := ports.Record{
		SessionID: row.SessionID,
		ParentID:  fromParent(row.ParentSessionID),
		Pipeline:  row.Pipeline,
		Date:      row.Date,
		Entries:   entries,
	}
	return &record, nil
}

func fromEntries(raw []byte) ([]ports.Entry, error) {
	entries := make([]ports.Entry, 0)
	if len(raw) == 0 {
		return entries, nil
	}
	if err := json.Unmarshal(raw, &entries); err != nil {
		return nil, err
	}
	return entries, nil
}

func fromParent(parentID *uuid.UUID) uuid.UUID {
	if parentID == nil {
		return uuid.Nil()
	}
	return *parentID
}
