package history

import (
	"context"
	"errors"
	"time"
	"uuid"
)

var ErrNotFound = errors.New("session history not found")

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

type Loader interface {
	// List gives the newest records first, at most limit of them.
	List(ctx context.Context, limit int32) ([]*Record, error)
	// Get gives one record. It returns ErrNotFound for an unknown session.
	Get(ctx context.Context, sessionID uuid.UUID) (*Record, error)
}

type HistoryService interface {
	Saver
	Loader
}
