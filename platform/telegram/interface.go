package telegram

type APIClient interface {
	SetWebhook(string) error
	DeleteWebhook() error
}
