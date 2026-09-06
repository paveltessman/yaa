package api

import (
	"github.com/paveltessman/yaa/pipelines/telegram/ports"
	"github.com/paveltessman/yaa/platform/settings"
	"github.com/paveltessman/yaa/platform/telegram"
)

type Deps struct {
	settings *settings.Settings
	tgClient ports.Webhooker
	dbRepo   ports.DBRepo
}

func NewDeps(s *settings.Settings, tgClient *telegram.Client, dbRepo ports.DBRepo) Deps {
	switch {
	case s == nil:
		panic("settings object is nil")
	case tgClient == nil:
		panic("tg client object is nil")
	case dbRepo == nil:
		panic("db repo object is nil")
	}

	d := Deps{
		settings: s,
		tgClient: tgClient,
		dbRepo:   dbRepo,
	}
	return d
}
