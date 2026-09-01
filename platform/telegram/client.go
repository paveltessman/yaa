package telegram

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/paveltessman/yaa/platform/network"
)

const apiURL = "https://api.telegram.org/bot"

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
	session := network.NewHTTPSession(apiURL + token)
	return session
}

func NewClient(session network.HTTPRequester) *Client {
	c := &Client{http: session}
	return c
}

func (c *Client) GetMe() (*GetMeResponse, error) {
	const path = "/getMe"
	resp := &GetMeResponse{}
	err := c.request(path, resp, nil)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *Client) SetWebhook(url string) error {
	const path = "/setWebhook"
	data := map[string]string{
		"url": url,
	}
	err := c.request(path, &baseResponse{}, data)
	return err
}

func (c *Client) DeleteWebhook() error {
	const path = "/deleteWebhook"
	err := c.request(path, &baseResponse{}, nil)
	return err
}

func (c *Client) request(path string, respModel oker, data map[string]string) error {
	body, err := json.Marshal(data)
	if err != nil {
		return err
	}
	r, err := c.http.PostBytes(path, network.ApplicationJson, body)
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
