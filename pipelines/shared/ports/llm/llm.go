package llm

import (
	"context"
	"errors"

	"time"
)

var ErrCompletionFailed = errors.New("completion failed")

type Role string

const (
	Assistant Role = "assistant"
	User      Role = "user"
)

type Message struct {
	Role Role      `json:"role"`
	Date time.Time `json:"-"`
	Text string    `json:"content"`
}

type CompletionParams struct {
	Model        string
	SystemPrompt string
	Input        []Message
}

type LLMService interface {
	Completion(ctx context.Context, params CompletionParams, response any) error
}
