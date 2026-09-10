package history

import (
	"encoding/json"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	. "github.com/paveltessman/yaa/pipelines/shared/ports/history"
	"github.com/paveltessman/yaa/platform/db/dbtest"
	"github.com/paveltessman/yaa/platform/db/internal/sqlc"
)

func newEntry(title string) Entry {
	entry := Entry{
		At:          time.Unix(1700000001, 0),
		Kind:        Log,
		Title:       title,
		Description: "what the pass did",
		Details:     []Detail{{Title: "the model", Body: "gemini-3.5-flash-lite"}},
	}
	return entry
}

func newRecord() *Record {
	record := Record{
		SessionID: uuid.NewV7(),
		ParentID:  uuid.NewV7(),
		Pipeline:  "telegram.updates",
		Date:      time.Unix(1700000000, 0),
		Entries:   []Entry{newEntry("the first entry")},
	}
	return &record
}

func read(t *testing.T, pool *pgxpool.Pool, sessionID uuid.UUID) sqlc.SessionHistory {
	t.Helper()

	const query = `SELECT session_id, parent_session_id, pipeline, date, entries
		FROM session_history WHERE session_id = $1`

	var got sqlc.SessionHistory
	err := pool.QueryRow(t.Context(), query, sessionID).Scan(
		&got.SessionID, &got.ParentSessionID, &got.Pipeline, &got.Date, &got.Entries,
	)
	if err != nil {
		t.Fatalf("reading the row back: %v", err)
	}
	return got
}

func count(t *testing.T, pool *pgxpool.Pool) int {
	t.Helper()

	var got int
	if err := pool.QueryRow(t.Context(), `SELECT count(*) FROM session_history`).Scan(&got); err != nil {
		t.Fatalf("counting the rows: %v", err)
	}
	return got
}

// entryCount asks postgres for the length of the json list. It fails when the
// column holds something that is no list.
func entryCount(t *testing.T, pool *pgxpool.Pool, sessionID uuid.UUID) int {
	t.Helper()

	const query = `SELECT jsonb_array_length(entries) FROM session_history WHERE session_id = $1`

	var got int
	if err := pool.QueryRow(t.Context(), query, sessionID).Scan(&got); err != nil {
		t.Fatalf("counting the entries: %v", err)
	}
	return got
}

func decode(t *testing.T, raw []byte) []Entry {
	t.Helper()

	var got []Entry
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("reading the entries back: %v", err)
	}
	return got
}

func TestSaveWritesEveryColumn(t *testing.T) {
	pool := dbtest.NewPool(t)
	repo := New(pool)
	record := newRecord()

	if err := repo.Save(t.Context(), record); err != nil {
		t.Fatalf("want no error, got %v", err)
	}

	got := read(t, pool, record.SessionID)
	if got.SessionID != record.SessionID {
		t.Errorf("session_id: want=%s, got=%s", record.SessionID, got.SessionID)
	}
	if got.ParentSessionID == nil {
		t.Fatalf("parent_session_id: want=%s, got a null", record.ParentID)
	}
	if *got.ParentSessionID != record.ParentID {
		t.Errorf("parent_session_id: want=%s, got=%s", record.ParentID, *got.ParentSessionID)
	}
	if got.Pipeline != record.Pipeline {
		t.Errorf("pipeline: want=%q, got=%q", record.Pipeline, got.Pipeline)
	}
	if !got.Date.Equal(record.Date) {
		t.Errorf("date: want=%s, got=%s", record.Date, got.Date)
	}
	if n := entryCount(t, pool, record.SessionID); n != 1 {
		t.Errorf("entries: want 1 entry, got %d", n)
	}
}

func TestSaveStoresAPassWithoutAParentAsNull(t *testing.T) {
	pool := dbtest.NewPool(t)
	repo := New(pool)
	record := newRecord()
	record.ParentID = uuid.Nil()

	if err := repo.Save(t.Context(), record); err != nil {
		t.Fatalf("want no error, got %v", err)
	}

	if got := read(t, pool, record.SessionID).ParentSessionID; got != nil {
		t.Errorf("parent_session_id: want a null, got %s", *got)
	}
}

func TestSaveKeepsEveryFieldOfAnEntry(t *testing.T) {
	pool := dbtest.NewPool(t)
	repo := New(pool)
	record := newRecord()

	if err := repo.Save(t.Context(), record); err != nil {
		t.Fatalf("want no error, got %v", err)
	}

	got := decode(t, read(t, pool, record.SessionID).Entries)
	if len(got) != 1 {
		t.Fatalf("want 1 entry, got %d", len(got))
	}
	first, want := got[0], record.Entries[0]
	if !first.At.Equal(want.At) {
		t.Errorf("at: want=%s, got=%s", want.At, first.At)
	}
	if first.Kind != want.Kind {
		t.Errorf("kind: want=%s, got=%s", want.Kind, first.Kind)
	}
	if first.Title != want.Title {
		t.Errorf("title: want=%q, got=%q", want.Title, first.Title)
	}
	if first.Description != want.Description {
		t.Errorf("description: want=%q, got=%q", want.Description, first.Description)
	}
	if len(first.Details) != len(want.Details) {
		t.Fatalf("details: want %d, got %d", len(want.Details), len(first.Details))
	}
	if first.Details[0] != want.Details[0] {
		t.Errorf("details: want=%+v, got=%+v", want.Details[0], first.Details[0])
	}
}

func TestSaveKeepsTheOrderOfTheEntries(t *testing.T) {
	pool := dbtest.NewPool(t)
	repo := New(pool)
	record := newRecord()
	record.Entries = []Entry{newEntry("first"), newEntry("second"), newEntry("third")}

	if err := repo.Save(t.Context(), record); err != nil {
		t.Fatalf("want no error, got %v", err)
	}

	got := decode(t, read(t, pool, record.SessionID).Entries)
	want := []string{"first", "second", "third"}
	if len(got) != len(want) {
		t.Fatalf("want %d entries, got %d", len(want), len(got))
	}
	for i := range want {
		if got[i].Title != want[i] {
			t.Errorf("entry %d: want=%q, got=%q", i, want[i], got[i].Title)
		}
	}
}

func TestSaveStoresAPassWithoutEntriesAsAnEmptyList(t *testing.T) {
	pool := dbtest.NewPool(t)
	repo := New(pool)

	for name, entries := range map[string][]Entry{"a nil list": nil, "an empty list": {}} {
		t.Run(name, func(t *testing.T) {
			record := newRecord()
			record.Entries = entries

			if err := repo.Save(t.Context(), record); err != nil {
				t.Fatalf("want no error, got %v", err)
			}
			if got := entryCount(t, pool, record.SessionID); got != 0 {
				t.Errorf("entries: want an empty list, got %d", got)
			}
		})
	}
}

func TestSaveReplacesTheRowOfTheSameSession(t *testing.T) {
	pool := dbtest.NewPool(t)
	repo := New(pool)
	record := newRecord()

	if err := repo.Save(t.Context(), record); err != nil {
		t.Fatalf("want no error, got %v", err)
	}

	repeat := newRecord()
	repeat.SessionID = record.SessionID
	repeat.Pipeline = "agent"
	repeat.Entries = []Entry{newEntry("first"), newEntry("second")}
	if err := repo.Save(t.Context(), repeat); err != nil {
		t.Fatalf("want no error on the repeat, got %v", err)
	}

	if got := count(t, pool); got != 1 {
		t.Errorf("want 1 row, got %d", got)
	}
	got := read(t, pool, record.SessionID)
	if got.Pipeline != repeat.Pipeline {
		t.Errorf("pipeline: want=%q, got=%q", repeat.Pipeline, got.Pipeline)
	}
	if n := entryCount(t, pool, record.SessionID); n != len(repeat.Entries) {
		t.Errorf("entries: want %d, got %d", len(repeat.Entries), n)
	}
}

func TestSaveStoresTheSessionOfEveryPass(t *testing.T) {
	pool := dbtest.NewPool(t)
	repo := New(pool)
	parent := newRecord()
	parent.ParentID = uuid.Nil()
	child := newRecord()
	child.ParentID = parent.SessionID
	child.Pipeline = "agent"

	for _, record := range []*Record{parent, child} {
		if err := repo.Save(t.Context(), record); err != nil {
			t.Fatalf("storing the %s pass: %v", record.Pipeline, err)
		}
	}

	if got := count(t, pool); got != 2 {
		t.Errorf("want 2 rows, got %d", got)
	}
	got := read(t, pool, child.SessionID).ParentSessionID
	if got == nil {
		t.Fatalf("parent_session_id: want=%s, got a null", parent.SessionID)
	}
	if *got != parent.SessionID {
		t.Errorf("parent_session_id: want=%s, got=%s", parent.SessionID, *got)
	}
}

func TestSaveSatisfiesTheHistoryService(t *testing.T) {
	pool := dbtest.NewPool(t)
	var service HistoryService = New(pool)
	record := newRecord()

	if err := service.Save(t.Context(), record); err != nil {
		t.Fatalf("want no error, got %v", err)
	}
	if got := count(t, pool); got != 1 {
		t.Errorf("want 1 row, got %d", got)
	}
}
