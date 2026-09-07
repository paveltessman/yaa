package settings

import (
	"net/url"
	"os"
)

const (
	defaultApiAddr     = "127.0.0.1:8080"
	defaultDatabaseURL = "postgres://yaa:yaa@localhost:5433/yaa?sslmode=disable"
)

type Settings struct {
	TgToken         string
	AnthropicApiKey string
	PublicHost      string
	ApiAddr         string
	DatabaseURL     string
}

func DatabaseURL() string {
	url := os.Getenv("DATABASE_URL")
	if len(url) == 0 {
		url = defaultDatabaseURL
	}
	return url
}

func NewSettings() Settings {
	TgToken := os.Getenv("TG_TOKEN")

	if len(TgToken) == 0 {
		panic("TG_TOKEN is not set")
	}

	AnthropicApiKey := os.Getenv("ANTHROPIC_API_KEY")
	if len(AnthropicApiKey) == 0 {
		panic("ANTHROPIC_API_KEY is not set")
	}

	ApiAddr := os.Getenv("API_ADDR")
	if len(ApiAddr) == 0 {
		ApiAddr = defaultApiAddr
	}

	PublicHost := os.Getenv("PUBLIC_HTTP_HOST")
	if _, err := url.Parse(PublicHost); err != nil {
		panic(err)
	}

	settings := Settings{
		TgToken:         TgToken,
		AnthropicApiKey: AnthropicApiKey,
		PublicHost:      PublicHost,
		ApiAddr:         ApiAddr,
		DatabaseURL:     DatabaseURL(),
	}
	return settings
}
