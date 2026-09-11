package ollama

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/paveltessman/yaa/pipelines/shared/ports/llm"
	"github.com/paveltessman/yaa/platform/network"
	testkit "github.com/paveltessman/yaa/platform/testkit/network"
)

var errTransport = errors.New("transport is down")

const testModel llm.Model = "qwen3:8b"

type reply struct {
	Text  string `json:"text" jsonschema:"description=the answer for the user"`
	Score int    `json:"score"`
}

func newTestClient(t *testing.T, resp string, err error) (*testkit.FakeRequester, *Client) {
	t.Helper()
	f := &testkit.FakeRequester{Resp: []byte(resp), Err: err}
	return f, NewClient(f)
}

func okResponse(text string) string {
	body, err := json.Marshal(completionResponse{
		Message:    message{Role: string(llm.Assistant), Content: text},
		Done:       true,
		DoneReason: doneStop,
	})
	if err != nil {
		panic(err)
	}
	return string(body)
}

func testParams() llm.CompletionParams {
	return llm.CompletionParams{
		Model:        testModel,
		SystemPrompt: "you are a bot",
		Input: []llm.Message{
			{Role: llm.User, Date: time.Unix(1700000000, 0), Text: "hello"},
		},
	}
}

func TestNewSession(t *testing.T) {
	cases := map[string]struct {
		host        string
		shouldPanic bool
	}{
		"plain host": {"http://localhost:11434", false},
		"empty host": {"", true},
		"no scheme":  {"localhost:11434", true},
	}
	for name, key := range cases {
		t.Run(name, func(t *testing.T) {
			defer func() {
				err := recover()
				if key.shouldPanic && err == nil {
					t.Errorf("NewSession accepted %s", name)
				}
				if !key.shouldPanic && err != nil {
					t.Errorf("want no panic, got %v", err)
				}
			}()

			if got := NewSession(key.host); got == nil && !key.shouldPanic {
				t.Errorf("want a session, got nil")
			}
		})
	}
}

func TestCompletionRequest(t *testing.T) {
	f, c := newTestClient(t, okResponse(`{"text":"hi","score":1}`), nil)

	if _, err := c.Completion(t.Context(), testParams(), &reply{}); err != nil {
		t.Fatalf("want no error, got %v", err)
	}

	if f.Calls != 1 {
		t.Errorf("want 1 call, got %d", f.Calls)
	}
	if want := chatPath; f.GotPath != want {
		t.Errorf("want=%q, got=%q", want, f.GotPath)
	}
	if f.GotType != network.ApplicationJson {
		t.Errorf("want content type %q, got %q", network.ApplicationJson, f.GotType)
	}

	want := `{"model":"qwen3:8b",` +
		`"messages":[{"role":"system","content":"you are a bot"},` +
		`{"role":"user","content":"hello"}],` +
		`"stream":false,` +
		`"format":{"properties":{"text":{"type":"string","description":"the answer for the user"},` +
		`"score":{"type":"integer"}},` +
		`"additionalProperties":false,"type":"object","required":["text","score"]},` +
		`"options":{"num_predict":16000}}`
	if string(f.GotBody) != want {
		t.Errorf("want body\n%s\ngot\n%s", want, f.GotBody)
	}
}

func TestCompletionDropsTheSystemPromptWhenEmpty(t *testing.T) {
	f, c := newTestClient(t, okResponse(`{"text":"hi","score":1}`), nil)
	params := testParams()
	params.SystemPrompt = ""

	if _, err := c.Completion(t.Context(), params, &reply{}); err != nil {
		t.Fatalf("want no error, got %v", err)
	}

	request := completionRequest{}
	if err := json.Unmarshal(f.GotBody, &request); err != nil {
		t.Fatalf("want valid json, got %v", err)
	}
	want := []message{{Role: string(llm.User), Content: "hello"}}
	if len(request.Messages) != len(want) || request.Messages[0] != want[0] {
		t.Errorf("want=%+v, got=%+v", want, request.Messages)
	}
}

func TestCompletionResult(t *testing.T) {
	cases := map[string]struct {
		resp string
		want reply
	}{
		"full object":  {okResponse(`{"text":"hi","score":7}`), reply{Text: "hi", Score: 7}},
		"empty object": {okResponse(`{}`), reply{}},
		// Older servers send no reason with the last chunk.
		"no done reason": {
			`{"message":{"role":"assistant","content":"{\"text\":\"hi\",\"score\":7}"},"done":true}`,
			reply{Text: "hi", Score: 7},
		},
	}
	for name, key := range cases {
		t.Run(name, func(t *testing.T) {
			_, c := newTestClient(t, key.resp, nil)

			got := reply{}
			if _, err := c.Completion(t.Context(), testParams(), &got); err != nil {
				t.Fatalf("want no error, got %v", err)
			}
			if got != key.want {
				t.Errorf("want=%+v, got=%+v", key.want, got)
			}
		})
	}
}

func TestCompletionRejectsABadResponseModel(t *testing.T) {
	text := "some text"
	cases := map[string]struct {
		model    any
		wantCall int
	}{
		// The schema comes from the model, so a bad model stops the request.
		"nil":    {nil, 0},
		"string": {&text, 0},
		"slice":  {&[]reply{}, 0},
		// A struct gives a good schema, but json.Unmarshal needs a pointer.
		"no pointer": {reply{}, 1},
	}
	for name, key := range cases {
		t.Run(name, func(t *testing.T) {
			f, c := newTestClient(t, okResponse(`{}`), nil)

			_, err := c.Completion(t.Context(), testParams(), key.model)

			if !errors.Is(err, llm.ErrCompletionFailed) {
				t.Fatalf("want ErrCompletionFailed, got %v", err)
			}
			if f.Calls != key.wantCall {
				t.Errorf("want %d calls, got %d", key.wantCall, f.Calls)
			}
		})
	}
}

func TestCompletionTransportError(t *testing.T) {
	_, c := newTestClient(t, "", errTransport)

	_, err := c.Completion(t.Context(), testParams(), &reply{})

	if !errors.Is(err, errTransport) {
		t.Errorf("want errTransport, got %v", err)
	}
	if !errors.Is(err, llm.ErrCompletionFailed) {
		t.Errorf("want ErrCompletionFailed, got %v", err)
	}
}

func TestCompletionBadJSON(t *testing.T) {
	cases := map[string]string{
		"not json":            "not json at all",
		"empty body":          "",
		"truncated json":      `{"done":true`,
		"array body":          `[1,2,3]`,
		"content is not json": okResponse("sorry, no json today"),
	}
	for name, resp := range cases {
		t.Run(name, func(t *testing.T) {
			_, c := newTestClient(t, resp, nil)

			_, err := c.Completion(t.Context(), testParams(), &reply{})

			if !errors.Is(err, llm.ErrCompletionFailed) {
				t.Fatalf("want ErrCompletionFailed, got %v", err)
			}
		})
	}
}

func TestCompletionUnusableAnswer(t *testing.T) {
	cases := map[string]string{
		"not done":      `{"message":{"role":"assistant","content":"{}"},"done":false}`,
		"cap reached":   `{"message":{"role":"assistant","content":"{\"text\":"},"done":true,"done_reason":"length"}`,
		"load only":     `{"message":{"role":"assistant","content":""},"done":true,"done_reason":"load"}`,
		"empty content": okResponse(""),
		"no message":    `{"done":true,"done_reason":"stop"}`,
	}
	for name, resp := range cases {
		t.Run(name, func(t *testing.T) {
			_, c := newTestClient(t, resp, nil)

			_, err := c.Completion(t.Context(), testParams(), &reply{})

			if !errors.Is(err, llm.ErrCompletionFailed) {
				t.Fatalf("want ErrCompletionFailed, got %v", err)
			}
		})
	}
}

func TestCompletionPassesTheContext(t *testing.T) {
	type key struct{}
	f, c := newTestClient(t, okResponse(`{}`), nil)
	ctx := context.WithValue(t.Context(), key{}, "marker")

	if _, err := c.Completion(ctx, testParams(), &reply{}); err != nil {
		t.Fatalf("want no error, got %v", err)
	}
	if f.GotCtx == nil {
		t.Fatal("want a context, got nil")
	}
	if got := f.GotCtx.Value(key{}); got != "marker" {
		t.Errorf("want=%q, got=%v", "marker", got)
	}
}
