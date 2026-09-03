package session

import (
	"github.com/paveltessman/yaa/pipelines/agent/models"
	"github.com/paveltessman/yaa/pipelines/shared"
)

type Session struct {
	shared.BaseSession
	Thread []models.Message
	Reply  string
}

func NewSession(thread []models.Message) *Session {
	s := Session{
		Thread: thread,
	}
	return &s
}
