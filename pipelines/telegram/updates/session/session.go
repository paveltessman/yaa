package session

import (
	"github.com/paveltessman/yaa/pipelines/shared"
	"github.com/paveltessman/yaa/pipelines/telegram/ports"
	"github.com/paveltessman/yaa/pipelines/telegram/updates/models"
)

var _ shared.Session = (*Session)(nil)

type Session struct {
	*shared.BaseSession
	RawUpdate []byte
	Update    *models.WhUpdate
	Message   *ports.Message
	Thread    []*ports.Message
}
