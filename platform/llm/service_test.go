package llm

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/paveltessman/yaa/pipelines/shared/ports/llm"
)

const (
	knownModel   llm.Model = "known"
	unknownModel llm.Model = "unknown"
)

type fakeBackend struct {
	calls  int
	gotCtx context.Context
	got    llm.CompletionParams
	err    error
}

func (f *fakeBackend) Completion(ctx context.Context, params llm.CompletionParams, _ any) error {
	f.calls++
	f.gotCtx = ctx
	f.got = params
	return f.err
}

func TestNewLLMService(t *testing.T) {
	cases := map[string]struct {
		backends    map[llm.Model]LLMBackend
		shouldPanic bool
	}{
		"one backend": {map[llm.Model]LLMBackend{knownModel: &fakeBackend{}}, false},
		"no backends": {map[llm.Model]LLMBackend{}, true},
		"nil map":     {nil, true},
		"nil backend": {map[llm.Model]LLMBackend{knownModel: nil}, true},
	}
	for name, key := range cases {
		t.Run(name, func(t *testing.T) {
			defer func() {
				err := recover()
				if key.shouldPanic && err == nil {
					t.Errorf("NewLLMService accepted %s", name)
				}
				if !key.shouldPanic && err != nil {
					t.Errorf("want no panic, got %v", err)
				}
			}()

			if got := NewLLMService(key.backends); got == nil && !key.shouldPanic {
				t.Errorf("want a service, got nil")
			}
		})
	}
}

func TestNewLLMServiceCopiesTheMap(t *testing.T) {
	backend := &fakeBackend{}
	backends := map[llm.Model]LLMBackend{knownModel: backend}

	s := NewLLMService(backends)
	delete(backends, knownModel)

	if err := s.Completion(t.Context(), llm.CompletionParams{Model: knownModel}, nil); err != nil {
		t.Fatalf("want no error, got %v", err)
	}
	if backend.calls != 1 {
		t.Errorf("want 1 call, got %d", backend.calls)
	}
}

func TestCompletionPicksTheBackend(t *testing.T) {
	backend := &fakeBackend{}
	other := &fakeBackend{}
	s := NewLLMService(map[llm.Model]LLMBackend{knownModel: backend, "other": other})
	params := llm.CompletionParams{Model: knownModel, SystemPrompt: "be short"}

	if err := s.Completion(t.Context(), params, nil); err != nil {
		t.Fatalf("want no error, got %v", err)
	}

	if backend.calls != 1 {
		t.Errorf("want 1 call, got %d", backend.calls)
	}
	if other.calls != 0 {
		t.Errorf("want no call on the other backend, got %d", other.calls)
	}
	if !reflect.DeepEqual(backend.got, params) {
		t.Errorf("want=%+v, got=%+v", params, backend.got)
	}
	if backend.gotCtx == nil {
		t.Error("want a context, got nil")
	}
}

func TestCompletionWithoutABackend(t *testing.T) {
	backend := &fakeBackend{}
	s := NewLLMService(map[llm.Model]LLMBackend{knownModel: backend})

	err := s.Completion(t.Context(), llm.CompletionParams{Model: unknownModel}, nil)

	if !errors.Is(err, llm.ErrCompletionFailed) {
		t.Fatalf("want ErrCompletionFailed, got %v", err)
	}
	if backend.calls != 0 {
		t.Errorf("want no call, got %d", backend.calls)
	}
}

func TestCompletionReturnsTheBackendError(t *testing.T) {
	wantErr := errors.New("backend is down")
	s := NewLLMService(map[llm.Model]LLMBackend{knownModel: &fakeBackend{err: wantErr}})

	err := s.Completion(t.Context(), llm.CompletionParams{Model: knownModel}, nil)

	if !errors.Is(err, wantErr) {
		t.Errorf("want the backend error, got %v", err)
	}
}
