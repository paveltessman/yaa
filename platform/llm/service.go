package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"maps"

	"github.com/paveltessman/yaa/pipelines/shared/ports/history"
	"github.com/paveltessman/yaa/pipelines/shared/ports/llm"
)

var _ llm.LLMService = (*LLMService)(nil)

type LLMBackend interface {
	Completion(ctx context.Context, params llm.CompletionParams, response any) ([]history.Detail, error)
}

type LLMService struct {
	backends map[llm.Model]LLMBackend
}

func NewLLMService(backends map[llm.Model]LLMBackend) *LLMService {
	if len(backends) == 0 {
		panic("at least one backend is required")
	}
	for model, backend := range backends {
		if backend == nil {
			panic(fmt.Sprintf("backend for model %q is nil", model))
		}
	}

	s := &LLMService{backends: maps.Clone(backends)}
	return s
}

func (s *LLMService) Completion(ctx context.Context, params llm.CompletionParams, response any) ([]history.Detail, error) {
	details := saveDetails(params)

	backend, ok := s.backends[params.Model]
	if !ok {
		return details, fmt.Errorf("%w: no backend for model: %q", llm.ErrCompletionFailed, params.Model)
	}
	backendDetails, err := backend.Completion(ctx, params, response)
	details = append(details, backendDetails...)
	return details, err
}

func saveDetails(params llm.CompletionParams) []history.Detail {
	details := make([]history.Detail, 0)

	inputBody, err := json.Marshal(params.Input)
	if err != nil {
		log.Printf("error while serializing llm imput: %v", err)
		return details
	}

	details = append(details, history.Detail{Title: "llm_input", Body: string(inputBody)})
	return details
}
