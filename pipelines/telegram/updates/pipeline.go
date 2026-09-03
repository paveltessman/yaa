package updates

import (
	"context"

	"github.com/paveltessman/yaa/pipelines/shared"
	"github.com/paveltessman/yaa/pipelines/telegram/updates/handlers"
	"github.com/paveltessman/yaa/pipelines/telegram/updates/models"
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

func NewChain(repo models.DBRepo, agentRunner handlers.AgentRunner) shared.Chain[*session.Session] {
	parseUpdate := shared.HandlerFunc[*session.Session](handlers.ParseUpdate)
	storeMessage := handlers.NewStoreMessage(repo)
	loadThread := handlers.NewLoadThread(repo)
	runAgent := handlers.NewRunAgent(agentRunner)

	chain := []shared.Handler[*session.Session]{
		parseUpdate,
		storeMessage,
		loadThread,
		runAgent,
	}
	return shared.NewChain(chain, errorHandler)
}
