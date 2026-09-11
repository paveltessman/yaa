package anthropic

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
	block := contentBlock{Type: textBlock, Text: text}
	body, err := json.Marshal(completionResponse{
		StopReason: stopEndTurn,
		Content:    []contentBlock{block},
	})
	if err != nil {
		panic(err)
	}
	return string(body)
}

func testParams() llm.CompletionParams {
	return llm.CompletionParams{
		Model:        llm.Opus5,
		SystemPrompt: "you are a bot",
		Input: []llm.Message{
			{Role: llm.User, Date: time.Unix(1700000000, 0), Text: "hello"},
		},
	}
}

func TestNewSession(t *testing.T) {
	cases := map[string]struct {
		apiKey      string
		shouldPanic bool
	}{
		"plain key": {"sk-ant-123", false},
		"empty key": {"", true},
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

			if got := NewSession(key.apiKey); got == nil && !key.shouldPanic {
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
	if want := messagesPath; f.GotPath != want {
		t.Errorf("want=%q, got=%q", want, f.GotPath)
	}
	if f.GotType != network.ApplicationJson {
		t.Errorf("want content type %q, got %q", network.ApplicationJson, f.GotType)
	}

	want := `{"model":"claude-opus-5","max_tokens":16000,"system":"you are a bot",` +
		`"messages":[{"role":"user","content":"hello"}],` +
		`"output_config":{"format":{"type":"json_schema","schema":{` +
		`"properties":{"text":{"type":"string","description":"the answer for the user"},` +
		`"score":{"type":"integer"}},` +
		`"additionalProperties":false,"type":"object","required":["text","score"]}}}}`
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

	var body map[string]any
	if err := json.Unmarshal(f.GotBody, &body); err != nil {
		t.Fatalf("want valid json, got %v", err)
	}
	if _, ok := body["system"]; ok {
		t.Errorf("want no system field, got %v", body["system"])
	}
}

func TestCompletionResult(t *testing.T) {
	cases := map[string]struct {
		resp string
		want reply
	}{
		"full object":  {okResponse(`{"text":"hi","score":7}`), reply{Text: "hi", Score: 7}},
		"empty object": {okResponse(`{}`), reply{}},
		"extra text block": {
			`{"stop_reason":"end_turn","content":[{"type":"thinking","thinking":"hmm"},` +
				`{"type":"text","text":"{\"text\":\"hi\",\"score\":7}"}]}`,
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
		"not json":         "not json at all",
		"empty body":       "",
		"truncated json":   `{"stop_reason":"end_turn"`,
		"array body":       `[1,2,3]`,
		"text is not json": okResponse("sorry, no json today"),
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
		"refusal":        `{"stop_reason":"refusal","content":[]}`,
		"max tokens":     `{"stop_reason":"max_tokens","content":[{"type":"text","text":"{\"text\":"}]}`,
		"no stop reason": `{"content":[{"type":"text","text":"{}"}]}`,
		"no text block":  `{"stop_reason":"end_turn","content":[{"type":"thinking","thinking":"hmm"}]}`,
		"no content":     `{"stop_reason":"end_turn"}`,
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
