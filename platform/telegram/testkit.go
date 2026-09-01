package telegram

var _ APIClient = (*FakeClient)(nil)

type FakeClient struct {
	Error      error
	CallCounts map[string]int
}

func (c *FakeClient) SetWebhook(params SetWebhookParams) error {
	c.CallCounts["SetWebhook"]++
	return c.Error
}

func (c *FakeClient) DeleteWebhook() error {
	c.CallCounts["DeleteWebhook"]++
	return c.Error
}
