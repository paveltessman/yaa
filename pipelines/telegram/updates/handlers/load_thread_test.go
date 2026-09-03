package handlers

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/paveltessman/yaa/pipelines/shared"
	"github.com/paveltessman/yaa/pipelines/telegram/updates/models"
	"github.com/paveltessman/yaa/pipelines/telegram/updates/session"
)

func newThread() []*models.Message {
	older := models.Message{
		ID:       9,
		ChatID:   40,
		ThreadID: 20,
		UserID:   30,
		Type:     models.FromUser,
		Date:     time.Unix(1699999999, 0),
		Text:     "hello there",
	}
	answer := models.Message{
		ID:       10,
		ChatID:   40,
		ThreadID: 20,
		UserID:   30,
		Type:     models.ToUser,
		Date:     time.Unix(1700000000, 0),
		Text:     "hi yourself",
	}
	return []*models.Message{&older, &answer}
}

func TestLoadThreadPutsTheThreadOnTheSession(t *testing.T) {
	thread := newThread()
	repo := models.FakeDBRepo{Thread: thread}
	h := NewLoadThread(&repo)
	s := newSessionWithMessage(newMessage())

	if err := h.Handle(context.Background(), s); err != nil {
		t.Fatalf("want no error, got %v", err)
	}
	if got := repo.CallCounts["LoadThread"]; got != 1 {
		t.Errorf("want 1 call, got %d", got)
	}
	if len(s.Thread) != len(thread) {
		t.Fatalf("want %d messages, got %d", len(thread), len(s.Thread))
	}
	for i, want := range thread {
		if got := s.Thread[i]; got != want {
			t.Errorf("message %d: want=%+v, got=%+v", i, want, got)
		}
	}
}

func TestLoadThreadAsksForTheChatAndThreadOfTheMessage(t *testing.T) {
	repo := models.FakeDBRepo{Thread: newThread()}
	h := NewLoadThread(&repo)
	message := newMessage()
	s := newSessionWithMessage(message)

	if err := h.Handle(context.Background(), s); err != nil {
		t.Fatalf("want no error, got %v", err)
	}
	if len(repo.LoadThreadCalls) != 1 {
		t.Fatalf("want 1 call, got %d", len(repo.LoadThreadCalls))
	}
	want := models.LoadThreadCall{ChatID: message.ChatID, ThreadID: message.ThreadID}
	if got := repo.LoadThreadCalls[0]; got != want {
		t.Errorf("want=%+v, got=%+v", want, got)
	}
}

func TestLoadThreadAsksForTheThreadOfAChatWithoutTopics(t *testing.T) {
	repo := models.FakeDBRepo{}
	h := NewLoadThread(&repo)
	message := newMessage()
	message.ThreadID = 0
	s := newSessionWithMessage(message)

	if err := h.Handle(context.Background(), s); err != nil {
		t.Fatalf("want no error, got %v", err)
	}
	if len(repo.LoadThreadCalls) != 1 {
		t.Fatalf("want 1 call, got %d", len(repo.LoadThreadCalls))
	}
	want := models.LoadThreadCall{ChatID: message.ChatID, ThreadID: 0}
	if got := repo.LoadThreadCalls[0]; got != want {
		t.Errorf("want=%+v, got=%+v", want, got)
	}
}

func TestLoadThreadKeepsMessageOnSession(t *testing.T) {
	repo := models.FakeDBRepo{Thread: newThread()}
	h := NewLoadThread(&repo)
	message := newMessage()
	s := newSessionWithMessage(message)

	if err := h.Handle(context.Background(), s); err != nil {
		t.Fatalf("want no error, got %v", err)
	}
	if s.Message != message {
		t.Errorf("want=%+v, got=%+v", message, s.Message)
	}
}

func TestLoadThreadPassesContext(t *testing.T) {
	type key struct{}
	repo := models.FakeDBRepo{Thread: newThread()}
	h := NewLoadThread(&repo)
	s := newSessionWithMessage(newMessage())
	ctx := context.WithValue(context.Background(), key{}, "marker")

	if err := h.Handle(ctx, s); err != nil {
		t.Fatalf("want no error, got %v", err)
	}
	if len(repo.Contexts) != 1 {
		t.Fatalf("want 1 context, got %d", len(repo.Contexts))
	}
	if got := repo.Contexts[0].Value(key{}); got != "marker" {
		t.Errorf("want=%q, got=%v", "marker", got)
	}
}

func TestLoadThreadRejectsSessionWithoutMessage(t *testing.T) {
	repo := models.FakeDBRepo{Thread: newThread()}
	h := NewLoadThread(&repo)
	s := newSessionWithMessage(nil)

	err := h.Handle(context.Background(), s)

	if err == nil {
		t.Fatal("want an error, got nil")
	}
	if !errors.Is(err, ErrLoadThread) {
		t.Errorf("want ErrLoadThread, got %v", err)
	}
	if got := repo.CallCounts["LoadThread"]; got != 0 {
		t.Errorf("want no call, got %d", got)
	}
	if s.Thread != nil {
		t.Errorf("want no thread on the session, got %+v", s.Thread)
	}
}

func TestLoadThreadWrapsRepoError(t *testing.T) {
	repoErr := errors.New("connection refused")
	repo := models.FakeDBRepo{Thread: newThread(), Error: repoErr}
	h := NewLoadThread(&repo)
	s := newSessionWithMessage(newMessage())

	err := h.Handle(context.Background(), s)

	if err == nil {
		t.Fatal("want an error, got nil")
	}
	if !errors.Is(err, ErrLoadThread) {
		t.Errorf("want ErrLoadThread, got %v", err)
	}
	if !errors.Is(err, repoErr) {
		t.Errorf("want the repo error, got %v", err)
	}
	if s.Thread != nil {
		t.Errorf("want no thread on the session, got %+v", s.Thread)
	}
}

func TestNewLoadThreadRejectsANilRepo(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("want a panic on a nil repo, got none")
		}
	}()

	NewLoadThread(nil)
}

func TestLoadThreadSatisfiesHandler(t *testing.T) {
	repo := models.FakeDBRepo{Thread: newThread()}
	var h shared.Handler[*session.Session] = NewLoadThread(&repo)
	s := newSessionWithMessage(newMessage())

	if err := h.Handle(context.Background(), s); err != nil {
		t.Fatalf("want no error, got %v", err)
	}
	if got := repo.CallCounts["LoadThread"]; got != 1 {
		t.Errorf("want 1 call, got %d", got)
	}
}
