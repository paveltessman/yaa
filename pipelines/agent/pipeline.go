package agent

import (
	"context"

	"github.com/paveltessman/yaa/pipelines/agent/handlers"
	"github.com/paveltessman/yaa/pipelines/agent/session"
	"github.com/paveltessman/yaa/pipelines/shared"
	"github.com/paveltessman/yaa/pipelines/shared/ports/llm"
)

func handleError(ctx context.Context, session *session.Session, err error) error {
	return err
}

func NewChain(service llm.LLMService) shared.Chain[*session.Session] {
	llmLoop := handlers.NewLlmLoop(service)

	chain := []shared.Handler[*session.Session]{
		llmLoop,
	}
	return shared.NewChain(chain, handleError)
}
