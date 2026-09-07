package llm

import (
	"context"
	"fmt"
	"maps"

	"github.com/paveltessman/yaa/pipelines/shared/ports/llm"
)

var _ llm.LLMService = (*LLMService)(nil)

type LLMBackend interface {
	Completion(ctx context.Context, params llm.CompletionParams, response any) error
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

func (s *LLMService) Completion(ctx context.Context, params llm.CompletionParams, response any) error {
	backend, ok := s.backends[params.Model]
	if !ok {
		return fmt.Errorf("%w: no backend for model: %q", llm.ErrCompletionFailed, params.Model)
	}
	return backend.Completion(ctx, params, response)
}
