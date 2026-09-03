package updates

import (
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

func NewChain(repo models.DBRepo) []shared.Handler[*session.Session] {
	parseUpdate := shared.HandlerFunc[*session.Session](handlers.ParseUpdate)
	storeMessage := handlers.NewStoreMessage(repo)

	chain := []shared.Handler[*session.Session]{
		parseUpdate,
		storeMessage,
	}
	return chain
}
