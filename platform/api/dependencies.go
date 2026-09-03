package api

import (
	"github.com/paveltessman/yaa/pipelines/telegram/updates/models"
	"github.com/paveltessman/yaa/platform/settings"
	"github.com/paveltessman/yaa/platform/telegram"
)

type Deps struct {
	settings *settings.Settings
	tgClient telegram.APIClient
	dbRepo   models.DBRepo
}

func NewDeps(s *settings.Settings, tgClient *telegram.Client, dbRepo models.DBRepo) Deps {
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
