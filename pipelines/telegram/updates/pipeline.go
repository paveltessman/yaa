package updates

import (
	"context"

	agent "github.com/paveltessman/yaa/pipelines/agent/session"

	"github.com/paveltessman/yaa/pipelines/shared"
	"github.com/paveltessman/yaa/pipelines/telegram/ports"
	"github.com/paveltessman/yaa/pipelines/telegram/updates/handlers"
	"github.com/paveltessman/yaa/pipelines/telegram/updates/session"
)

func NewSession(update []byte) *session.Session {
	session := session.Session{
		BaseSession: shared.NewSession(),
		RawUpdate:   update,
	}
	return &session

}

func errorHandler(ctx context.Context, session *session.Session, err error) error {
	return err
}

func NewChain(
	client ports.Sender,
	repo ports.DBRepo,
	agentRunner handlers.AgentRunner,
	agentChain shared.Chain[*agent.Session],
) shared.Chain[*session.Session] {
	parseUpdate := shared.HandlerFunc[*session.Session](handlers.ParseUpdate)
	storeMessage := handlers.NewStoreMessage(repo)
	loadThread := handlers.NewLoadThread(repo)
	runAgent := handlers.NewRunAgent(agentRunner, agentChain)
	sendReply := handlers.NewSendReply(client)
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
