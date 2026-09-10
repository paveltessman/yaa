package shared

import (
	"context"
	"errors"
	"io"
	"log"
	"os"
	"testing"
	"time"
	"uuid"

	"github.com/paveltessman/yaa/pipelines/shared/ports/history"
	testkit "github.com/paveltessman/yaa/platform/testkit/history"
)

const testPipelineName = "test"

func quietLog(t *testing.T) {
	t.Helper()
	log.SetOutput(io.Discard)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })
}

func newEntry(title string) history.Entry {
	entry := history.Entry{
		At:          time.Unix(1700000000, 0),
		Kind:        history.Log,
		Title:       title,
		Description: "what the pass did",
		Details:     []history.Detail{{Title: "detail", Body: "body"}},
	}
	return entry
}

// newChain makes a chain of one handler that runs fn.
func newChain(fn func(*BaseSession) error) Chain[*BaseSession] {
	handler := HandlerFunc[*BaseSession](func(_ context.Context, session *BaseSession) error {
		return fn(session)
	})
	errorHandler := func(_ context.Context, _ *BaseSession, err error) error { return err }
	return NewChain([]Handler[*BaseSession]{handler}, errorHandler)
}

func noopChain() Chain[*BaseSession] {
	return newChain(func(*BaseSession) error { return nil })
}

func savedRecord(t *testing.T, saver *testkit.FakeHistoryService) *history.Record {
	t.Helper()

	if saver.Calls != 1 {
		t.Fatalf("want 1 save, got %d", saver.Calls)
	}
	return saver.Records[0]
}

func TestPipelineSavesTheRecordOfThePass(t *testing.T) {
	saver := testkit.FakeHistoryService{}
	pipeline := NewPipeline(testPipelineName, &saver, noopChain())
	parentID := uuid.NewV7()
	session := NewSession(parentID)

	if err := pipeline(context.Background(), session); err != nil {
		t.Fatalf("want no error, got %v", err)
	}

	got := savedRecord(t, &saver)
	if got.Pipeline != testPipelineName {
		t.Errorf("pipeline: want=%q, got=%q", testPipelineName, got.Pipeline)
	}
	if got.SessionID != session.ID() {
		t.Errorf("session id: want=%s, got=%s", session.ID(), got.SessionID)
	}
	if got.ParentID != parentID {
		t.Errorf("parent id: want=%s, got=%s", parentID, got.ParentID)
	}
	if !got.Date.Equal(session.Date()) {
		t.Errorf("date: want=%s, got=%s", session.Date(), got.Date)
	}
	if len(got.Entries) != 0 {
		t.Errorf("want no entry, got %+v", got.Entries)
	}
}

func TestPipelineSavesAPassWithoutAParent(t *testing.T) {
	saver := testkit.FakeHistoryService{}
	pipeline := NewPipeline(testPipelineName, &saver, noopChain())

	if err := pipeline(context.Background(), NewSession(uuid.Nil())); err != nil {
		t.Fatalf("want no error, got %v", err)
	}

	if got := savedRecord(t, &saver).ParentID; got != uuid.Nil() {
		t.Errorf("parent id: want the nil uuid, got %s", got)
	}
}

func TestPipelineSavesTheEntriesOfTheSession(t *testing.T) {
	saver := testkit.FakeHistoryService{}
	chain := newChain(func(session *BaseSession) error {
		session.AppendHistory(newEntry("first"))
		session.AppendHistory(newEntry("second"))
		return nil
	})
	pipeline := NewPipeline(testPipelineName, &saver, chain)

	if err := pipeline(context.Background(), NewSession(uuid.Nil())); err != nil {
		t.Fatalf("want no error, got %v", err)
	}

	got := savedRecord(t, &saver).Entries
	want := []history.Entry{newEntry("first"), newEntry("second")}
	if len(got) != len(want) {
		t.Fatalf("want %d entries, got %d", len(want), len(got))
	}
	for i := range want {
		if got[i].Title != want[i].Title {
			t.Errorf("entry %d: title: want=%q, got=%q", i, want[i].Title, got[i].Title)
		}
		if got[i].Kind != want[i].Kind {
			t.Errorf("entry %d: kind: want=%s, got=%s", i, want[i].Kind, got[i].Kind)
		}
	}
}

func TestPipelinePassesTheContextToTheSaver(t *testing.T) {
	type key struct{}
	saver := testkit.FakeHistoryService{}
	pipeline := NewPipeline(testPipelineName, &saver, noopChain())
	ctx := context.WithValue(context.Background(), key{}, "marker")

	if err := pipeline(ctx, NewSession(uuid.Nil())); err != nil {
		t.Fatalf("want no error, got %v", err)
	}

	if len(saver.Contexts) != 1 {
		t.Fatalf("want 1 save, got %d", len(saver.Contexts))
	}
	if got := saver.Contexts[0].Value(key{}); got != "marker" {
		t.Errorf("want=%q, got=%v", "marker", got)
	}
}

func TestPipelineSavesAfterAChainError(t *testing.T) {
	chainErr := errors.New("the handler failed")
	saver := testkit.FakeHistoryService{}
	chain := newChain(func(session *BaseSession) error {
		session.AppendHistory(newEntry("first"))
		return chainErr
	})
	pipeline := NewPipeline(testPipelineName, &saver, chain)

	err := pipeline(context.Background(), NewSession(uuid.Nil()))

	if !errors.Is(err, chainErr) {
		t.Errorf("want the chain error, got %v", err)
	}
	if got := savedRecord(t, &saver).Entries; len(got) != 1 {
		t.Errorf("want 1 entry, got %d", len(got))
	}
}

func TestPipelineKeepsTheChainErrorWhenTheSaveFails(t *testing.T) {
	quietLog(t)
	chainErr := errors.New("the handler failed")
	saver := testkit.FakeHistoryService{Error: errors.New("the save failed")}
	pipeline := NewPipeline(testPipelineName, &saver, newChain(func(*BaseSession) error { return chainErr }))

	err := pipeline(context.Background(), NewSession(uuid.Nil()))

	if !errors.Is(err, chainErr) {
		t.Errorf("want the chain error, got %v", err)
	}
}

func TestPipelineHidesTheSaveErrorFromAPassThatWorks(t *testing.T) {
	quietLog(t)
	saver := testkit.FakeHistoryService{Error: errors.New("the save failed")}
	pipeline := NewPipeline(testPipelineName, &saver, noopChain())

	if err := pipeline(context.Background(), NewSession(uuid.Nil())); err != nil {
		t.Errorf("want no error, got %v", err)
	}
}

func TestNewPipelineRejectsABadArgument(t *testing.T) {
	cases := map[string]struct {
		name  string
		saver history.Saver
	}{
		"empty name": {"", &testkit.FakeHistoryService{}},
		"nil saver":  {testPipelineName, nil},
	}

	for name, argument := range cases {
		t.Run(name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Errorf("want a panic on a %s, got none", name)
				}
			}()

			NewPipeline(argument.name, argument.saver, noopChain())
		})
	}
}
