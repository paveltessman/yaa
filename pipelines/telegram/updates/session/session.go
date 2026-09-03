package session

import (
	"github.com/paveltessman/yaa/pipelines/shared"
	"github.com/paveltessman/yaa/pipelines/telegram/updates/models"
	"github.com/paveltessman/yaa/platform/telegram"
)

var _ shared.Session = (*Session)(nil)

type Session struct {
	*shared.BaseSession
	RawUpdate []byte
	Update    *telegram.Update
	Message   *models.Message
	Thread    []*models.Message
}
