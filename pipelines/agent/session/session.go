package session

import (
	"github.com/paveltessman/yaa/pipelines/shared"
	"github.com/paveltessman/yaa/pipelines/shared/ports/llm"
)

type Session struct {
	shared.BaseSession
	Thread []llm.Message
	Reply  string
}

func NewSession(thread []llm.Message) *Session {
	s := Session{
		Thread: thread,
	}
	return &s
}
