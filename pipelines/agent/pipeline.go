package agent

import (
	"context"

	"github.com/paveltessman/yaa/pipelines/agent/handlers"
	"github.com/paveltessman/yaa/pipelines/agent/session"
	"github.com/paveltessman/yaa/pipelines/shared"
)

func handleError(ctx context.Context, session *session.Session, err error) error {
	return err
}

func NewChain() shared.Chain[*session.Session] {
	chain := []shared.Handler[*session.Session]{
		shared.HandlerFunc[*session.Session](handlers.LlmLoop),
	}
	return shared.NewChain(chain, handleError)
}
