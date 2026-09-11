package anthropic

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
	apiURL     = "https://api.anthropic.com"
	apiVersion = "2023-06-01"

	messagesPath = "/v1/messages"

	requestTimeout = 2 * time.Minute
	maxTokens      = 16000
)

const (
	jsonSchemaFormat = "json_schema"
	textBlock        = "text"
	stopEndTurn      = "end_turn"
)

type outputFormat struct {
	Type   string             `json:"type"`
	Schema *jsonschema.Schema `json:"schema"`
}

type outputConfig struct {
	Format outputFormat `json:"format"`
}

type completionRequest struct {
	Model        llm.Model     `json:"model"`
	MaxTokens    int           `json:"max_tokens"`
	System       string        `json:"system,omitempty"`
	Messages     []llm.Message `json:"messages"`
	OutputConfig outputConfig  `json:"output_config"`
}

type contentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type completionResponse struct {
	StopReason string         `json:"stop_reason"`
	Content    []contentBlock `json:"content"`
}

func (r *completionResponse) text() (string, error) {
	for _, block := range r.Content {
		if block.Type == textBlock {
			return block.Text, nil
		}
	}
	return "", fmt.Errorf("%w: the response carries no text block", llm.ErrCompletionFailed)
}

type Client struct {
	http      network.HTTPRequester
	reflector *jsonschema.Reflector
}

func NewSession(apiKey string) *network.HTTPSession {
	if len(apiKey) == 0 {
		panic("empty api key is not allowed")
	}
	headers := map[string]string{
		"x-api-key":         apiKey,
		"anthropic-version": apiVersion,
	}
	session := network.NewHTTPSessionWithHeaders(apiURL, requestTimeout, headers)
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

	details = append(details, history.Detail{Title: "model", Body: string(params.Model)})
	details = append(details, history.Detail{Title: "system_prompt", Body: params.SystemPrompt})

	request := completionRequest{
		Model:     params.Model,
		MaxTokens: maxTokens,
		System:    params.SystemPrompt,
		Messages:  params.Input,
		OutputConfig: outputConfig{
			Format: outputFormat{Type: jsonSchemaFormat, Schema: schema},
		},
	}

	body, err := json.Marshal(request)
	if err != nil {
		return details, fmt.Errorf("%w: %w", llm.ErrCompletionFailed, err)
	}

	raw, err := c.http.PostBytes(ctx, messagesPath, network.ApplicationJson, body)
	details = append(details, history.Detail{Title: "raw_response", Body: string(raw)})
	if err != nil {
		return details, fmt.Errorf("%w: %w", llm.ErrCompletionFailed, err)
	}

	resp := completionResponse{}
	if err := json.Unmarshal(raw, &resp); err != nil {
		return details, fmt.Errorf("%w: %w", llm.ErrCompletionFailed, err)
	}
	// Any other stop reason means the model refused or ran out of room, and
	// then the text does not match the schema.
	if resp.StopReason != stopEndTurn {
		return details, fmt.Errorf("%w: the model stopped with %q", llm.ErrCompletionFailed, resp.StopReason)
	}

	text, err := resp.text()
	if err != nil {
		return details, err
	}

	details = append(details, history.Detail{Title: "response_text", Body: text})

	if err := json.Unmarshal([]byte(text), response); err != nil {
		return details, fmt.Errorf("%w: %w", llm.ErrCompletionFailed, err)
	}
	return details, nil
}

func (c *Client) schemaFor(response any) (*jsonschema.Schema, error) {
	if response == nil {
		return nil, fmt.Errorf("%w: the response model is nil", llm.ErrCompletionFailed)
	}

	schema := c.reflector.Reflect(response)
	if schema.Type != "object" {
		return nil, fmt.Errorf("%w: the response model must be a struct, got %q", llm.ErrCompletionFailed, schema.Type)
	}
	// The API takes the schema alone, without the dialect marker.
	schema.Version = ""
	return schema, nil
}
