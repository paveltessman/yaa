package telegram

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/paveltessman/yaa/platform/network"
)

var _ APIClient = (*Client)(nil)

const apiURL = "https://api.telegram.org/bot"

// requestTimeout caps one call to the telegram api.
const requestTimeout = 10 * time.Second

var ErrTelegramAPIFailed = errors.New("telegram api failed")

type oker interface {
	ok() bool
}

type baseResponse struct {
	Ok bool
}

func (r *baseResponse) ok() bool {
	return r.Ok
}

type GetMeResponse struct {
	baseResponse
	Result struct {
		ID    int64
		IsBot bool `json:"is_bot"`
	}
}

type Client struct {
	http network.HTTPRequester
}

func NewSession(token string) *network.HTTPSession {
	if len(token) == 0 {
		panic("empty token is not allowed")
	}
	session := network.NewHTTPSession(apiURL+token, requestTimeout)
	return session
}

func NewClient(session network.HTTPRequester) *Client {
	c := &Client{http: session}
	return c
}

func (c *Client) GetMe(ctx context.Context) (*GetMeResponse, error) {
	const path = "/getMe"
	resp := &GetMeResponse{}
	err := c.request(ctx, path, resp, nil)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *Client) SetWebhook(ctx context.Context, params SetWebhookParams) error {
	const path = "/setWebhook"
	err := c.request(ctx, path, &baseResponse{}, params)
	return err
}

func (c *Client) DeleteWebhook(ctx context.Context) error {
	const path = "/deleteWebhook"
	err := c.request(ctx, path, &baseResponse{}, nil)
	return err
}

func (c *Client) request(ctx context.Context, path string, respModel oker, data any) error {
	body, err := json.Marshal(data)
	if err != nil {
		return err
	}
	r, err := c.http.PostBytes(ctx, path, network.ApplicationJson, body)
	if err != nil {
		return err
	}

	err = json.Unmarshal(r, respModel)
	if err != nil {
		return err
	}

	if !respModel.ok() {
		return fmt.Errorf("%w: %s", ErrTelegramAPIFailed, string(r))
	}
	return nil
}
