package updates

import (
	"context"
	"uuid"

	agent "github.com/paveltessman/yaa/pipelines/agent/session"
	"github.com/paveltessman/yaa/pipelines/shared/ports/history"

	"github.com/paveltessman/yaa/pipelines/shared"
	"github.com/paveltessman/yaa/pipelines/telegram/ports"
	"github.com/paveltessman/yaa/pipelines/telegram/updates/handlers"
	"github.com/paveltessman/yaa/pipelines/telegram/updates/session"
)

// pipelineName names the passes of this pipeline in the history.
const pipelineName = "telegram.updates"

func NewSession(update []byte) *session.Session {
	session := session.Session{
		BaseSession: shared.NewSession(uuid.Nil()),
		RawUpdate:   update,
	}
	return &session

}

func errorHandler(ctx context.Context, session *session.Session, err error) error {
	return err
}

func NewChain(
	tg ports.Sender,
	repo ports.DBRepo,
	agentPipeline shared.Pipeline[*agent.Session],
) shared.Chain[*session.Session] {
	parseUpdate := shared.HandlerFunc[*session.Session](handlers.ParseUpdate)
	storeMessage := handlers.NewStoreMessage(repo)
	loadThread := handlers.NewLoadThread(repo)
	runAgent := handlers.NewRunAgent(agentPipeline)
	sendReply := handlers.NewSendReply(tg)
	storeReply := handlers.NewStoreReply(repo)

	chain := []shared.Handler[*session.Session]{
		parseUpdate,
		storeMessage,
		loadThread,
		runAgent,
		sendReply,
		storeReply,
	}
	return shared.NewChain(chain, errorHandler)
}

func NewPipeline(
	tg ports.Sender,
	repo ports.DBRepo,
	agentPipeline shared.Pipeline[*agent.Session],
	history history.Saver,
) shared.Pipeline[*session.Session] {
	chain := NewChain(tg, repo, agentPipeline)
	pipeline := shared.NewPipeline(pipelineName, history, chain)
	return pipeline
}
