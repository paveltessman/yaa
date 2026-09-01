package settings

import (
	"net/url"
	"os"
)

const defaultApiAddr = "127.0.0.1:8080"

type Settings struct {
	TgToken    string
	PublicHost string
	ApiAddr    string
}

func NewSettings() Settings {
	TgToken := os.Getenv("TG_TOKEN")

	if len(TgToken) == 0 {
		panic("TG_TOKEN is not set")
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
		TgToken:    TgToken,
		PublicHost: PublicHost,
		ApiAddr:    ApiAddr,
	}
	return settings
}
