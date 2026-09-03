package handlers

import (
	"context"
	"log"

	pipelines "github.com/paveltessman/yaa/pipelines/agent"
	agent "github.com/paveltessman/yaa/pipelines/agent/session"

	"github.com/paveltessman/yaa/pipelines/shared"
	"github.com/paveltessman/yaa/pipelines/telegram/updates/session"
)

type AgentRunner shared.Runner[*agent.Session]

type RunAgent struct {
	agentRunner AgentRunner
}

func NewRunAgent(agentRunner AgentRunner) RunAgent {
	h := RunAgent{
		agentRunner: agentRunner,
	}
	return h
}

func (h RunAgent) Handle(ctx context.Context, session *session.Session) error {
	s := agent.NewSession(nil)
	chain := pipelines.NewChain()
	err := h.agentRunner(ctx, s, chain)
	if err == nil {
		log.Println(s.Reply)
	}
	return err
}
