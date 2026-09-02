package session

import (
	"github.com/paveltessman/yaa/pipelines/shared"
)

var _ shared.Session = (*Session)(nil)

type Session struct {
	*shared.BaseSession
	RawUpdate []byte
}
