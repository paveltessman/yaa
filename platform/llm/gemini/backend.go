package gemini

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/invopop/jsonschema"

	"github.com/paveltessman/yaa/pipelines/shared/ports/history"
	"github.com/paveltessman/yaa/pipelines/shared/ports/llm"
	"github.com/paveltessman/yaa/platform/network"
)

const (
	apiURL = "https://generativelanguage.googleapis.com"

	completionPath = "/v1beta/models/%s:generateContent" // The %s is a model string

	requestTimeout  = 30 * time.Second
	maxOutputTokens = 16000
)

const (
	jsonMimeType = "application/json"
	modelRole    = "model"
	finishStop   = "STOP"
)

type part struct {
	Text    string `json:"text"`
	Thought bool   `json:"thought,omitempty"`
}

type content struct {
	Role  string `json:"role,omitempty"`
	Parts []part `json:"parts"`
}

type generationConfig struct {
	ResponseMimeType   string             `json:"responseMimeType"`
	ResponseJsonSchema *jsonschema.Schema `json:"responseJsonSchema"`
	MaxOutputTokens    int                `json:"maxOutputTokens"`
}

type completionRequest struct {
	SystemInstruction *content         `json:"systemInstruction,omitempty"`
	Contents          []content        `json:"contents"`
	GenerationConfig  generationConfig `json:"generationConfig"`
}

type candidate struct {
	Content      content `json:"content"`
	FinishReason string  `json:"finishReason"`
}

type promptFeedback struct {
	BlockReason string `json:"blockReason"`
}

type completionResponse struct {
	Candidates     []candidate     `json:"candidates"`
	PromptFeedback *promptFeedback `json:"promptFeedback"`
}

func (r *completionResponse) text() (string, error) {
	if r.PromptFeedback != nil && r.PromptFeedback.BlockReason != "" {
		return "", fmt.Errorf("%w: the api blocked the prompt with %q", llm.ErrCompletionFailed, r.PromptFeedback.BlockReason)
	}
	if len(r.Candidates) == 0 {
		return "", fmt.Errorf("%w: the response carries no candidate", llm.ErrCompletionFailed)
	}

	first := r.Candidates[0]
	// Any other reason means the model refused or ran out of room, and then
	// the text does not match the schema.
	if first.FinishReason != finishStop {
		return "", fmt.Errorf("%w: the model stopped with %q", llm.ErrCompletionFailed, first.FinishReason)
	}

	text := strings.Builder{}
	for _, p := range first.Content.Parts {
		if p.Thought {
			continue
		}
		text.WriteString(p.Text)
	}
	if text.Len() == 0 {
		return "", fmt.Errorf("%w: the response carries no text part", llm.ErrCompletionFailed)
	}
	return text.String(), nil
}

type Client struct {
	http      network.HTTPRequester
	reflector *jsonschema.Reflector
}

func NewSession(apiKey string) *network.HTTPSession {
	if len(apiKey) == 0 {
		panic("empty api key is not allowed")
	}
	headers := map[string]string{"x-goog-api-key": apiKey}
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
	if params.Model == "" {
		return details, fmt.Errorf("%w: the model is empty", llm.ErrCompletionFailed)
	}

	details = append(details, history.Detail{Title: "model", Body: string(params.Model)})

	request := completionRequest{
		Contents: toContents(params.Input),
		GenerationConfig: generationConfig{
			ResponseMimeType:   jsonMimeType,
			ResponseJsonSchema: schema,
			MaxOutputTokens:    maxOutputTokens,
		},
	}
	if params.SystemPrompt != "" {
		request.SystemInstruction = &content{Parts: []part{{Text: params.SystemPrompt}}}
	}

	body, err := json.Marshal(request)
	if err != nil {
		return details, fmt.Errorf("%w: %w", llm.ErrCompletionFailed, err)
	}

	details = append(details, history.Detail{Title: "backend_request", Body: string(body)})

	path := fmt.Sprintf(completionPath, params.Model)
	raw, err := c.http.PostBytes(ctx, path, network.ApplicationJson, body)
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

	details = append(details, history.Detail{Title: "response_text", Body: text})

	if err := json.Unmarshal([]byte(text), response); err != nil {
		return details, fmt.Errorf("%w: %w", llm.ErrCompletionFailed, err)
	}
	return details, nil
}

func toContents(input []llm.Message) []content {
	contents := make([]content, 0, len(input))
	for _, message := range input {
		contents = append(contents, content{Role: toRole(message.Role), Parts: []part{{Text: message.Text}}})
	}
	return contents
}

func toRole(role llm.Role) string {
	if role == llm.Assistant {
		return modelRole
	}
	return "user"
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
