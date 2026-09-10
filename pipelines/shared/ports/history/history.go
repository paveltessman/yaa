package history

import (
	"context"
	"time"
	"uuid"
)

type Kind string

const (
	Log Kind = "log"
	LLM Kind = "llm"
)

type Detail struct {
	Title string
	Body  string
}

type Entry struct {
	At          time.Time
	Kind        Kind
	Title       string
	Description string
	Details     []Detail
}

type Saver interface {
	Save(ctx context.Context, sessionId uuid.UUID, entries []Entry) error
}

type HistoryService interface {
	Saver
}
