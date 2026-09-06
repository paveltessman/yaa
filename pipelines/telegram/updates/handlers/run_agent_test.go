package handlers

import (
	"context"
	"errors"
	"testing"

	agent "github.com/paveltessman/yaa/pipelines/agent/session"
	"github.com/paveltessman/yaa/pipelines/shared"
	"github.com/paveltessman/yaa/pipelines/telegram/updates/session"
)

func fakeAgentRunner(reply string, err error, contexts *[]context.Context) AgentRunner {
	runner := func(ctx context.Context, s *agent.Session, chain shared.Chain[*agent.Session]) error {
		if contexts != nil {
			*contexts = append(*contexts, ctx)
		}
		if err != nil {
			return err
		}
		s.Reply = reply
		return nil
	}
	return runner
}

func TestRunAgentPutsTheReplyOnTheSession(t *testing.T) {
	h := NewRunAgent(fakeAgentRunner("hi yourself", nil, nil))
	s := newSessionWithMessage(newMessage())

	if err := h.Handle(context.Background(), s); err != nil {
		t.Fatalf("want no error, got %v", err)
	}
	if want := "hi yourself"; s.Reply != want {
		t.Errorf("want=%q, got=%q", want, s.Reply)
	}
}

func TestRunAgentPassesContext(t *testing.T) {
	type key struct{}
	var contexts []context.Context
	h := NewRunAgent(fakeAgentRunner("hi yourself", nil, &contexts))
	s := newSessionWithMessage(newMessage())
	ctx := context.WithValue(context.Background(), key{}, "marker")

	if err := h.Handle(ctx, s); err != nil {
		t.Fatalf("want no error, got %v", err)
	}
	if len(contexts) != 1 {
		t.Fatalf("want 1 context, got %d", len(contexts))
	}
	if got := contexts[0].Value(key{}); got != "marker" {
		t.Errorf("want=%q, got=%v", "marker", got)
	}
}

func TestRunAgentReturnsRunnerError(t *testing.T) {
	runnerErr := errors.New("agent is down")
	h := NewRunAgent(fakeAgentRunner("hi yourself", runnerErr, nil))
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
	var h shared.Handler[*session.Session] = NewRunAgent(fakeAgentRunner("hi yourself", nil, nil))
	s := newSessionWithMessage(newMessage())

	if err := h.Handle(context.Background(), s); err != nil {
		t.Fatalf("want no error, got %v", err)
	}
	if s.Reply == "" {
		t.Error("want a reply on the session, got none")
	}
}
