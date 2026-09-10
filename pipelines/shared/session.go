package shared

import (
	"time"
	"uuid"

	"github.com/paveltessman/yaa/pipelines/shared/ports/history"
)

var _ Session = (*BaseSession)(nil)

type Session interface {
	ID() uuid.UUID
	Date() time.Time
	History() []history.Entry
}

type BaseSession struct {
	id      uuid.UUID
	date    time.Time
	history []history.Entry
}

func (s *BaseSession) ID() uuid.UUID {
	return s.id
}

func (s *BaseSession) Date() time.Time {
	return s.date
}

func (s *BaseSession) AppendHistory(entry history.Entry) {
	s.history = append(s.history, entry)
}

func (s *BaseSession) History() []history.Entry {
	out := make([]history.Entry, 0, len(s.history))
	out = append(out, s.history...)
	return out
}

func NewSession() *BaseSession {
	session := BaseSession{
		id:   uuid.NewV7(),
		date: time.Now(),
	}
	return &session
}
