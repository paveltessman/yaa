package shared

import (
	"time"
	"uuid"
)

var _ Session = (*BaseSession)(nil)

type Session interface {
	ID() uuid.UUID
	Date() time.Time
}

type BaseSession struct {
	id   uuid.UUID
	date time.Time
}

func (s *BaseSession) ID() uuid.UUID {
	return s.id
}

func (s *BaseSession) Date() time.Time {
	return s.date
}

func NewSession() *BaseSession {
	session := BaseSession{
		id:   uuid.NewV7(),
		date: time.Now(),
	}
	return &session
}
