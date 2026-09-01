package telegram

type SetWebhookParams struct {
	URL            string   `json:"url"`
	AllowedUpdates []string `json:"allowed_updates"`
}

type APIClient interface {
	SetWebhook(SetWebhookParams) error
	DeleteWebhook() error
}
