package handlers

import (
	"context"

	agent "github.com/paveltessman/yaa/pipelines/agent/session"

	"github.com/paveltessman/yaa/pipelines/shared"
	"github.com/paveltessman/yaa/pipelines/shared/ports/llm"
	"github.com/paveltessman/yaa/pipelines/telegram/ports"
	"github.com/paveltessman/yaa/pipelines/telegram/updates/session"
)

type RunAgent struct {
	pipeline shared.Pipeline[*agent.Session]
}

func NewRunAgent(agentPipeline shared.Pipeline[*agent.Session]) RunAgent {
	if agentPipeline == nil {
		panic("agent runner object is nil")
	}

	h := RunAgent{
		pipeline: agentPipeline,
	}
	return h
}

func toRole(messageType ports.MessageType) llm.Role {
	if messageType == ports.ToUser {
		return llm.Assistant
	}
	return llm.User
}

func toThread(messages []*ports.Message) []llm.Message {
	thread := make([]llm.Message, 0, len(messages))
	for _, message := range messages {
		thread = append(thread, llm.Message{
			Role: toRole(message.Type),
			Date: message.Date,
			Text: message.Text,
		})
	}
	return thread
}

func (h RunAgent) Handle(ctx context.Context, session *session.Session) error {
	s := agent.NewSession(toThread(session.Thread))

	err := h.pipeline(ctx, s)
	if err != nil {
		return err
	}

	session.Reply = s.Reply
	return nil
}
