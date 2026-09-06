package ports

import "context"

type SetWebhookParams struct {
	URL            string   `json:"url"`
	AllowedUpdates []string `json:"allowed_updates"`
}

type Webhooker interface {
	SetWebhook(context.Context, SetWebhookParams) error
	DeleteWebhook(context.Context) error
}

type SendMessageParams struct {
	ChatID   int64  `json:"chat_id"`
	ThreadID int64  `json:"message_thread_id,omitempty"`
	Text     string `json:"text"`
}

type Sender interface {
	SendMessage(context.Context, SendMessageParams) (*Message, error)
}
