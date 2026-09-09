package session

import (
	"github.com/paveltessman/yaa/pipelines/shared"
	"github.com/paveltessman/yaa/pipelines/shared/ports/llm"
)

var _ shared.Session = (*Session)(nil)

type Session struct {
	*shared.BaseSession
	Thread []llm.Message
	Reply  string
}

func NewSession(thread []llm.Message) *Session {
	s := Session{
		BaseSession: shared.NewSession(),
		Thread:      thread,
	}
	return &s
}
