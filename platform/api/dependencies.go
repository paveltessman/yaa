package api

import (
	"github.com/paveltessman/yaa/platform/settings"
	"github.com/paveltessman/yaa/platform/telegram"
)

type Deps struct {
	settings *settings.Settings
	tgClient telegram.APIClient
}

func NewDeps(s *settings.Settings, tgClient *telegram.Client) Deps {
	switch {
	case s == nil:
		panic("settings object is nil")
	case tgClient == nil:
		panic("tg client object is nil")
	}

	d := Deps{
		settings: s,
		tgClient: tgClient,
	}
	return d
}
