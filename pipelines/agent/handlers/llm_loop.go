package handlers

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/paveltessman/yaa/pipelines/agent/session"
	"github.com/paveltessman/yaa/pipelines/shared/ports/history"
	"github.com/paveltessman/yaa/pipelines/shared/ports/llm"
)

var ErrLlmLoop = errors.New("unable to generate reply")

const llmLoopModel = llm.Gemini35FlashLite

const systemPrompt = `You are yaa, a personal assistant.
You talk to the user in a telegram chat.
Answer the last message of the user.
Keep the answer short and plain, telegram shows no formatting.`

type reply struct {
	Text string `json:"text" jsonschema:"description=the answer for the user"`
}

type LlmLoop struct {
	service llm.LLMService
}

func NewLlmLoop(service llm.LLMService) LlmLoop {
	if service == nil {
		panic("llm service object is nil")
	}

	h := LlmLoop{service: service}
	return h
}

func (h LlmLoop) Handle(ctx context.Context, session *session.Session) error {
	if len(session.Thread) == 0 {
		return fmt.Errorf("%w: session has no thread", ErrLlmLoop)
	}

	params := llm.CompletionParams{
		Model:        llmLoopModel,
		SystemPrompt: systemPrompt,
		Input:        session.Thread,
	}

	answer := reply{}
	details, err := h.service.Completion(ctx, params, &answer)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrLlmLoop, err)
	}
	if answer.Text == "" {
		return fmt.Errorf("%w: the model returned no text", ErrLlmLoop)
	}

	session.Reply = answer.Text
	session.AppendHistory(historyEntry(details))
	return nil
}

func historyEntry(details []history.Detail) history.Entry {
	entry := history.Entry{
		At:          time.Now(),
		Kind:        history.LLM,
		Title:       "llm step",
		Description: "",
		Details:     details,
	}
	return entry
}
