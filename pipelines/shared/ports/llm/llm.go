package llm

import (
	"context"
	"errors"

	"time"

	"github.com/paveltessman/yaa/pipelines/shared/ports/history"
)

type Model string

const (
	Opus5   Model = "claude-opus-5"
	Sonnet5 Model = "claude-sonnet-5"
	Haiku45 Model = "claude-haiku-4-5"

	Gemini35FlashLite Model = "gemini-3.5-flash-lite"
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
	Model        Model
	SystemPrompt string
	Input        []Message
}

type LLMService interface {
	Completion(ctx context.Context, params CompletionParams, response any) ([]history.Detail, error)
}
