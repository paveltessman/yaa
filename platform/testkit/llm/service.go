package llm

import (
	"context"
	"encoding/json"

	"github.com/paveltessman/yaa/pipelines/shared/ports/history"
	"github.com/paveltessman/yaa/pipelines/shared/ports/llm"
)

var _ llm.LLMService = (*FakeLLMService)(nil)

type FakeLLMService struct {
	Response any
	Err      error

	Calls     int
	GotCtx    context.Context
	GotParams llm.CompletionParams
}

func (f *FakeLLMService) Completion(ctx context.Context, params llm.CompletionParams, response any) ([]history.Detail, error) {
	details := make([]history.Detail, 0)
	f.Calls++
	f.GotCtx = ctx
	f.GotParams = params
	if f.Err != nil {
		return details, f.Err
	}
	if f.Response == nil || response == nil {
		return details, nil
	}

	body, err := json.Marshal(f.Response)
	if err != nil {
		return details, err
	}
	return details, json.Unmarshal(body, response)
}
