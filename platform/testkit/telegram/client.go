package telegram

import (
	"context"

	"github.com/paveltessman/yaa/pipelines/telegram/ports"
)

var _ ports.Webhooker = (*FakeClient)(nil)
var _ ports.Sender = (*FakeClient)(nil)

type FakeClient struct {
	Error            error
	SentMessage      *ports.Message
	CallCounts       map[string]int
	Contexts         []context.Context
	SendMessageCalls []ports.SendMessageParams
}

func (c *FakeClient) SetWebhook(ctx context.Context, params ports.SetWebhookParams) error {
	c.record(ctx, "SetWebhook")
	return c.Error
}

func (c *FakeClient) DeleteWebhook(ctx context.Context) error {
	c.record(ctx, "DeleteWebhook")
	return c.Error
}

func (c *FakeClient) SendMessage(ctx context.Context, params ports.SendMessageParams) (*ports.Message, error) {
	c.record(ctx, "SendMessage")
	c.SendMessageCalls = append(c.SendMessageCalls, params)
	if c.Error != nil {
		return nil, c.Error
	}
	return c.SentMessage, nil
}

func (c *FakeClient) record(ctx context.Context, name string) {
	if c.CallCounts == nil {
		c.CallCounts = make(map[string]int)
	}
	c.CallCounts[name]++
	c.Contexts = append(c.Contexts, ctx)
}
