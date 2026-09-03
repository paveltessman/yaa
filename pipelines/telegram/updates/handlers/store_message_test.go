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

func newMessage() *models.Message {
	message := models.Message{
		ID:       10,
		ChatID:   40,
		ThreadID: 20,
		UserID:   30,
		Type:     models.FromUser,
		Date:     time.Unix(1700000000, 0),
		Text:     "hello",
	}
	return &message
}

func newSessionWithMessage(message *models.Message) *session.Session {
	s := newSession(`{"message":{"message_id":10,"date":1700000000}}`)
	s.Message = message
	return s
}

func TestStoreMessageStoresSessionMessage(t *testing.T) {
	repo := models.FakeDBRepo{}
	h := NewStoreMessage(&repo)
	message := newMessage()
	s := newSessionWithMessage(message)

	if err := h.Handle(context.Background(), s); err != nil {
		t.Fatalf("want no error, got %v", err)
	}
	if got := repo.CallCounts["StoreMessage"]; got != 1 {
		t.Errorf("want 1 call, got %d", got)
	}
	if len(repo.Messages) != 1 {
		t.Fatalf("want 1 stored message, got %d", len(repo.Messages))
	}
	if got := repo.Messages[0]; got != message {
		t.Errorf("want=%+v, got=%+v", message, got)
	}
}

func TestStoreMessageKeepsMessageOnSession(t *testing.T) {
	repo := models.FakeDBRepo{}
	h := NewStoreMessage(&repo)
	message := newMessage()
	s := newSessionWithMessage(message)

	if err := h.Handle(context.Background(), s); err != nil {
		t.Fatalf("want no error, got %v", err)
	}
	if s.Message != message {
		t.Errorf("want=%+v, got=%+v", message, s.Message)
	}
}

func TestStoreMessagePassesContext(t *testing.T) {
	type key struct{}
	repo := models.FakeDBRepo{}
	h := NewStoreMessage(&repo)
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

func TestStoreMessageRejectsSessionWithoutMessage(t *testing.T) {
	repo := models.FakeDBRepo{}
	h := NewStoreMessage(&repo)
	s := newSessionWithMessage(nil)

	err := h.Handle(context.Background(), s)

	if err == nil {
		t.Fatal("want an error, got nil")
	}
	if !errors.Is(err, ErrStoreMessage) {
		t.Errorf("want ErrStoreMessage, got %v", err)
	}
	if got := repo.CallCounts["StoreMessage"]; got != 0 {
		t.Errorf("want no call, got %d", got)
	}
}

func TestStoreMessageWrapsRepoError(t *testing.T) {
	repoErr := errors.New("connection refused")
	repo := models.FakeDBRepo{Error: repoErr}
	h := NewStoreMessage(&repo)
	s := newSessionWithMessage(newMessage())

	err := h.Handle(context.Background(), s)

	if err == nil {
		t.Fatal("want an error, got nil")
	}
	if !errors.Is(err, ErrStoreMessage) {
		t.Errorf("want ErrStoreMessage, got %v", err)
	}
	if !errors.Is(err, repoErr) {
		t.Errorf("want the repo error, got %v", err)
	}
}

func TestNewStoreMessageRejectsANilRepo(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("want a panic on a nil repo, got none")
		}
	}()

	NewStoreMessage(nil)
}

func TestStoreMessageSatisfiesHandler(t *testing.T) {
	repo := models.FakeDBRepo{}
	var h shared.Handler[*session.Session] = NewStoreMessage(&repo)
	s := newSessionWithMessage(newMessage())

	if err := h.Handle(context.Background(), s); err != nil {
		t.Fatalf("want no error, got %v", err)
	}
	if got := repo.CallCounts["StoreMessage"]; got != 1 {
		t.Errorf("want 1 call, got %d", got)
	}
}
