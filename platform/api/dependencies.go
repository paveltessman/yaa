package api

import (
	"github.com/paveltessman/yaa/pipelines/shared/ports/llm"
	"github.com/paveltessman/yaa/pipelines/telegram/ports"
	"github.com/paveltessman/yaa/platform/settings"
	"github.com/paveltessman/yaa/platform/telegram"
)

type Deps struct {
	settings   *settings.Settings
	tgClient   ports.Client
	dbRepo     ports.DBRepo
	llmService llm.LLMService
}

func NewDeps(
	s *settings.Settings,
	tgClient *telegram.Client,
	dbRepo ports.DBRepo,
	llmService llm.LLMService,
) Deps {
	switch {
	case s == nil:
		panic("settings object is nil")
	case tgClient == nil:
		panic("tg client object is nil")
	case dbRepo == nil:
		panic("db repo object is nil")
	case llmService == nil:
		panic("llm service object is nil")
	}

	d := Deps{
		settings:   s,
		tgClient:   tgClient,
		dbRepo:     dbRepo,
		llmService: llmService,
	}
	return d
}
