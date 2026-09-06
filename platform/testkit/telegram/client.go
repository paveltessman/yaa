package telegram

import (
	"context"

	"github.com/paveltessman/yaa/pipelines/telegram/ports"
)

var _ ports.Webhooker = (*FakeClient)(nil)

type FakeClient struct {
	Error      error
	CallCounts map[string]int
	Contexts   []context.Context
}

func (c *FakeClient) SetWebhook(ctx context.Context, params ports.SetWebhookParams) error {
	c.record(ctx, "SetWebhook")
	return c.Error
}

func (c *FakeClient) DeleteWebhook(ctx context.Context) error {
	c.record(ctx, "DeleteWebhook")
	return c.Error
}

func (c *FakeClient) record(ctx context.Context, name string) {
	if c.CallCounts == nil {
		c.CallCounts = make(map[string]int)
	}
	c.CallCounts[name]++
	c.Contexts = append(c.Contexts, ctx)
}
