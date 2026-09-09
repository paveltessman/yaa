package handlers

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/paveltessman/yaa/pipelines/agent/session"
	"github.com/paveltessman/yaa/pipelines/shared"
	"github.com/paveltessman/yaa/pipelines/shared/ports/llm"
	testkit "github.com/paveltessman/yaa/platform/testkit/llm"
)

func newThread() []llm.Message {
	older := llm.Message{
		Role: llm.User,
		Date: time.Unix(1699999999, 0),
		Text: "hello there",
	}
	answer := llm.Message{
		Role: llm.Assistant,
		Date: time.Unix(1700000000, 0),
		Text: "hi yourself",
	}
	return []llm.Message{older, answer}
}

func newAnswer(text string) map[string]string {
	return map[string]string{"text": text}
}

func newService(text string, err error) *testkit.FakeLLMService {
	return &testkit.FakeLLMService{Response: newAnswer(text), Err: err}
}

func TestLlmLoopPutsTheReplyOnTheSession(t *testing.T) {
	service := newService("the answer", nil)
	h := NewLlmLoop(service)
	s := session.NewSession(newThread())

	if err := h.Handle(context.Background(), s); err != nil {
		t.Fatalf("want no error, got %v", err)
	}
	if service.Calls != 1 {
		t.Errorf("want 1 call, got %d", service.Calls)
	}
	if want := "the answer"; s.Reply != want {
		t.Errorf("want=%q, got=%q", want, s.Reply)
	}
}

func TestLlmLoopSendsTheThreadAsInput(t *testing.T) {
	service := newService("the answer", nil)
	h := NewLlmLoop(service)
	thread := newThread()
	s := session.NewSession(thread)

	if err := h.Handle(context.Background(), s); err != nil {
		t.Fatalf("want no error, got %v", err)
	}

	got := service.GotParams.Input
	if len(got) != len(thread) {
		t.Fatalf("want %d messages, got %d", len(thread), len(got))
	}
	for i, want := range thread {
		if got[i] != want {
			t.Errorf("message %d: want=%+v, got=%+v", i, want, got[i])
		}
	}
}

func TestLlmLoopSendsTheModelAndTheSystemPrompt(t *testing.T) {
	service := newService("the answer", nil)
	h := NewLlmLoop(service)
	s := session.NewSession(newThread())

	if err := h.Handle(context.Background(), s); err != nil {
		t.Fatalf("want no error, got %v", err)
	}
	if got := service.GotParams.Model; got != llmLoopModel {
		t.Errorf("want=%q, got=%q", llmLoopModel, got)
	}
	if got := service.GotParams.SystemPrompt; got != systemPrompt {
		t.Errorf("want=%q, got=%q", systemPrompt, got)
	}
}

func TestLlmLoopPassesTheContext(t *testing.T) {
	type key struct{}
	service := newService("the answer", nil)
	h := NewLlmLoop(service)
	s := session.NewSession(newThread())
	ctx := context.WithValue(context.Background(), key{}, "marker")

	if err := h.Handle(ctx, s); err != nil {
		t.Fatalf("want no error, got %v", err)
	}
	if got := service.GotCtx.Value(key{}); got != "marker" {
		t.Errorf("want=%q, got=%v", "marker", got)
	}
}

func TestLlmLoopWithAnEmptyThread(t *testing.T) {
	service := newService("the answer", nil)
	h := NewLlmLoop(service)
	s := session.NewSession(nil)

	err := h.Handle(context.Background(), s)

	if !errors.Is(err, ErrLlmLoop) {
		t.Fatalf("want ErrLlmLoop, got %v", err)
	}
	if service.Calls != 0 {
		t.Errorf("want no call, got %d", service.Calls)
	}
}

func TestLlmLoopWithAnEmptyAnswer(t *testing.T) {
	service := newService("", nil)
	h := NewLlmLoop(service)
	s := session.NewSession(newThread())

	err := h.Handle(context.Background(), s)

	if !errors.Is(err, ErrLlmLoop) {
		t.Fatalf("want ErrLlmLoop, got %v", err)
	}
	if s.Reply != "" {
		t.Errorf("want no reply on the session, got %q", s.Reply)
	}
}

func TestLlmLoopReturnsTheServiceError(t *testing.T) {
	serviceErr := errors.New("the backend is down")
	h := NewLlmLoop(newService("the answer", serviceErr))
	s := session.NewSession(newThread())

	err := h.Handle(context.Background(), s)

	if !errors.Is(err, serviceErr) {
		t.Errorf("want the service error, got %v", err)
	}
	if !errors.Is(err, ErrLlmLoop) {
		t.Errorf("want ErrLlmLoop, got %v", err)
	}
	if s.Reply != "" {
		t.Errorf("want no reply on the session, got %q", s.Reply)
	}
}

func TestLlmLoopSatisfiesHandler(t *testing.T) {
	var h shared.Handler[*session.Session] = NewLlmLoop(newService("the answer", nil))
	s := session.NewSession(newThread())

	if err := h.Handle(context.Background(), s); err != nil {
		t.Fatalf("want no error, got %v", err)
	}
	if s.Reply == "" {
		t.Error("want a reply on the session, got none")
	}
}

func TestNewLlmLoopRejectsANilService(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("want a panic on a nil service, got none")
		}
	}()

	NewLlmLoop(nil)
}
