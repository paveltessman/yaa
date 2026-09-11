package ollama

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/invopop/jsonschema"

	"github.com/paveltessman/yaa/pipelines/shared/ports/history"
	"github.com/paveltessman/yaa/pipelines/shared/ports/llm"
	"github.com/paveltessman/yaa/platform/network"
)

const (
	chatPath       = "/api/chat"
	requestTimeout = 5 * time.Minute
	numPredict     = 16000
)

const (
	systemRole = "system"
	doneStop   = "stop"
)

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type options struct {
	NumPredict int `json:"num_predict"`
}

type completionRequest struct {
	Model    llm.Model          `json:"model"`
	Messages []message          `json:"messages"`
	Stream   bool               `json:"stream"`
	Format   *jsonschema.Schema `json:"format"`
	Options  options            `json:"options"`
}

type completionResponse struct {
	Message    message `json:"message"`
	Done       bool    `json:"done"`
	DoneReason string  `json:"done_reason"`
}

func (r *completionResponse) text() (string, error) {
	if !r.Done {
		return "", fmt.Errorf("%w: the response is not done", llm.ErrCompletionFailed)
	}
	if r.DoneReason != "" && r.DoneReason != doneStop {
		return "", fmt.Errorf("%w: the model stopped with %q", llm.ErrCompletionFailed, r.DoneReason)
	}
	if r.Message.Content == "" {
		return "", fmt.Errorf("%w: the response carries no content", llm.ErrCompletionFailed)
	}
	return r.Message.Content, nil
}

type Client struct {
	http      network.HTTPRequester
	reflector *jsonschema.Reflector
}

func NewSession(host string) *network.HTTPSession {
	session := network.NewHTTPSession(host, requestTimeout)
	return session
}

func NewClient(session network.HTTPRequester) *Client {
	c := &Client{
		http: session,
		reflector: &jsonschema.Reflector{
			DoNotReference: true,
			ExpandedStruct: true,
			Anonymous:      true,
		},
	}
	return c
}

func (c *Client) Completion(ctx context.Context, params llm.CompletionParams, response any) ([]history.Detail, error) {
	details := make([]history.Detail, 0)

	schema, err := c.schemaFor(response)
	if err != nil {
		return details, err
	}

	request := completionRequest{
		Model:    params.Model,
		Messages: toMessages(params),
		Stream:   false,
		Format:   schema,
		Options:  options{NumPredict: numPredict},
	}

	body, err := json.Marshal(request)
	if err != nil {
		return details, fmt.Errorf("%w: %w", llm.ErrCompletionFailed, err)
	}

	details = append(details, history.Detail{Title: "backend_request", Body: string(body)})

	raw, err := c.http.PostBytes(ctx, chatPath, network.ApplicationJson, body)
	details = append(details, history.Detail{Title: "raw_response", Body: string(raw)})
	if err != nil {
		return details, fmt.Errorf("%w: %w", llm.ErrCompletionFailed, err)
	}

	resp := completionResponse{}
	if err := json.Unmarshal(raw, &resp); err != nil {
		return details, fmt.Errorf("%w: %w", llm.ErrCompletionFailed, err)
	}

	text, err := resp.text()
	if err != nil {
		return details, err
	}

	details = append(details, history.Detail{Title: "response_text", Body: string(text)})

	if err := json.Unmarshal([]byte(text), response); err != nil {
		return details, fmt.Errorf("%w: %w", llm.ErrCompletionFailed, err)
	}
	return details, nil
}

func toMessages(params llm.CompletionParams) []message {
	messages := make([]message, 0, len(params.Input)+1)
	if params.SystemPrompt != "" {
		messages = append(messages, message{Role: systemRole, Content: params.SystemPrompt})
	}
	for _, input := range params.Input {
		messages = append(messages, message{Role: string(input.Role), Content: input.Text})
	}
	return messages
}

func (c *Client) schemaFor(response any) (*jsonschema.Schema, error) {
	if response == nil {
		return nil, fmt.Errorf("%w: the response model is nil", llm.ErrCompletionFailed)
	}

	schema := c.reflector.Reflect(response)
	if schema.Type != "object" {
		return nil, fmt.Errorf("%w: the response model must be a struct, got %q", llm.ErrCompletionFailed, schema.Type)
	}
	schema.Version = ""
	return schema, nil
}
