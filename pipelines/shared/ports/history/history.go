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
	Title string `json:"title"`
	Body  string `json:"body"`
}

type Entry struct {
	At          time.Time `json:"at"`
	Kind        Kind      `json:"kind"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Details     []Detail  `json:"details,omitempty"`
}

type Record struct {
	SessionID uuid.UUID
	ParentID  uuid.UUID
	Pipeline  string
	Date      time.Time
	Entries   []Entry
}

type Saver interface {
	Save(ctx context.Context, record *Record) error
}

type HistoryService interface {
	Saver
}
