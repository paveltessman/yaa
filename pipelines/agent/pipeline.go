package agent

import (
	"github.com/paveltessman/yaa/pipelines/agent/handlers"
	"github.com/paveltessman/yaa/pipelines/agent/session"
	"github.com/paveltessman/yaa/pipelines/shared"
)

func NewChain() []shared.Handler[*session.Session] {
	chain := []shared.Handler[*session.Session]{
		shared.HandlerFunc[*session.Session](handlers.LlmLoop),
	}
	return chain
}
