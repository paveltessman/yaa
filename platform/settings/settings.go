package settings

import "os"

const defaultApiAddr = "127.0.0.1:8080"

type Settings struct {
	TgToken string
	ApiAddr string
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

	settings := Settings{
		TgToken: TgToken,
		ApiAddr: ApiAddr,
	}
	return settings
}
