package telegram

import "context"

type SetWebhookParams struct {
	URL            string   `json:"url"`
	AllowedUpdates []string `json:"allowed_updates"`
}

type APIClient interface {
	SetWebhook(context.Context, SetWebhookParams) error
	DeleteWebhook(context.Context) error
}
