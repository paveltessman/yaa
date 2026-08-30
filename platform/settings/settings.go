package settings

import "os"

type Settings struct {
	TgToken string
}

func NewSettings() Settings {
	TgToken := os.Getenv("TG_TOKEN")

	if len(TgToken) == 0 {
		panic("TG_TOKEN is not set")
	}
	settings := Settings{
		TgToken: TgToken,
	}
	return settings
}
