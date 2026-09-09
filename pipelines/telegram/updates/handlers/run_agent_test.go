package handlers

import (
	"context"
	"errors"
	"testing"

	agent "github.com/paveltessman/yaa/pipelines/agent/session"
	"github.com/paveltessman/yaa/pipelines/shared"
	"github.com/paveltessman/yaa/pipelines/shared/ports/llm"
	. "github.com/paveltessman/yaa/pipelines/telegram/ports"
	"github.com/paveltessman/yaa/pipelines/telegram/updates/session"
)

// agentCall holds the arguments of one runner call.
type agentCall struct {
	ctx     context.Context
	session *agent.Session
}

func fakeAgentRunner(reply string, err error, calls *[]agentCall) AgentRunner {
	runner := func(ctx context.Context, s *agent.Session, chain shared.Chain[*agent.Session]) error {
		if calls != nil {
			*calls = append(*calls, agentCall{ctx: ctx, session: s})
		}
		if err != nil {
			return err
		}
		s.Reply = reply
		return nil
	}
	return runner
}

func replyChain(reply string) shared.Chain[*agent.Session] {
	handler := shared.HandlerFunc[*agent.Session](func(ctx context.Context, s *agent.Session) error {
		s.Reply = reply
		return nil
	})
	errorHandler := func(ctx context.Context, s *agent.Session, err error) error { return err }
	return shared.NewChain([]shared.Handler[*agent.Session]{handler}, errorHandler)
}

func emptyChain() shared.Chain[*agent.Session] {
	errorHandler := func(ctx context.Context, s *agent.Session, err error) error { return err }
	return shared.NewChain(nil, errorHandler)
}

func newSessionWithThread(thread []*Message) *session.Session {
	s := newSessionWithMessage(newMessage())
	s.Thread = thread
	return s
}

func TestRunAgentPutsTheReplyOnTheSession(t *testing.T) {
	h := NewRunAgent(fakeAgentRunner("hi yourself", nil, nil), emptyChain())
	s := newSessionWithMessage(newMessage())

	if err := h.Handle(context.Background(), s); err != nil {
		t.Fatalf("want no error, got %v", err)
	}
	if want := "hi yourself"; s.Reply != want {
		t.Errorf("want=%q, got=%q", want, s.Reply)
	}
}

func TestRunAgentRunsTheChainItWasBuiltWith(t *testing.T) {
	h := NewRunAgent(shared.Run, replyChain("from the chain"))
	s := newSessionWithMessage(newMessage())

	if err := h.Handle(context.Background(), s); err != nil {
		t.Fatalf("want no error, got %v", err)
	}
	if want := "from the chain"; s.Reply != want {
		t.Errorf("want=%q, got=%q", want, s.Reply)
	}
}

func TestRunAgentPassesTheThreadToTheAgent(t *testing.T) {
	var calls []agentCall
	h := NewRunAgent(fakeAgentRunner("hi yourself", nil, &calls), emptyChain())
	thread := newThread()
	s := newSessionWithThread(thread)

	if err := h.Handle(context.Background(), s); err != nil {
		t.Fatalf("want no error, got %v", err)
	}
	if len(calls) != 1 {
		t.Fatalf("want 1 call, got %d", len(calls))
	}

	got := calls[0].session.Thread
	if len(got) != len(thread) {
		t.Fatalf("want %d messages, got %d", len(thread), len(got))
	}
	want := []llm.Message{
		{Role: llm.User, Date: thread[0].Date, Text: thread[0].Text},
		{Role: llm.Assistant, Date: thread[1].Date, Text: thread[1].Text},
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("message %d: want=%+v, got=%+v", i, want[i], got[i])
		}
	}
}

func TestRunAgentPassesAnEmptyThread(t *testing.T) {
	var calls []agentCall
	h := NewRunAgent(fakeAgentRunner("hi yourself", nil, &calls), emptyChain())
	s := newSessionWithThread(nil)

	if err := h.Handle(context.Background(), s); err != nil {
		t.Fatalf("want no error, got %v", err)
	}
	if len(calls) != 1 {
		t.Fatalf("want 1 call, got %d", len(calls))
	}
	if got := calls[0].session.Thread; len(got) != 0 {
		t.Errorf("want no messages, got %+v", got)
	}
}

func TestRunAgentPassesContext(t *testing.T) {
	type key struct{}
	var calls []agentCall
	h := NewRunAgent(fakeAgentRunner("hi yourself", nil, &calls), emptyChain())
	s := newSessionWithMessage(newMessage())
	ctx := context.WithValue(context.Background(), key{}, "marker")

	if err := h.Handle(ctx, s); err != nil {
		t.Fatalf("want no error, got %v", err)
	}
	if len(calls) != 1 {
		t.Fatalf("want 1 call, got %d", len(calls))
	}
	if got := calls[0].ctx.Value(key{}); got != "marker" {
		t.Errorf("want=%q, got=%v", "marker", got)
	}
}

func TestRunAgentReturnsRunnerError(t *testing.T) {
	runnerErr := errors.New("agent is down")
	h := NewRunAgent(fakeAgentRunner("hi yourself", runnerErr, nil), emptyChain())
	s := newSessionWithMessage(newMessage())

	err := h.Handle(context.Background(), s)

	if !errors.Is(err, runnerErr) {
		t.Errorf("want the runner error, got %v", err)
	}
	if s.Reply != "" {
		t.Errorf("want no reply on the session, got %q", s.Reply)
	}
}

func TestRunAgentSatisfiesHandler(t *testing.T) {
	var h shared.Handler[*session.Session] = NewRunAgent(fakeAgentRunner("hi yourself", nil, nil), emptyChain())
	s := newSessionWithMessage(newMessage())

	if err := h.Handle(context.Background(), s); err != nil {
		t.Fatalf("want no error, got %v", err)
	}
	if s.Reply == "" {
		t.Error("want a reply on the session, got none")
	}
}

func TestNewRunAgentRejectsANilRunner(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("want a panic on a nil runner, got none")
		}
	}()

	NewRunAgent(nil, emptyChain())
}
