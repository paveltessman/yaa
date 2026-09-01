package api

import (
	"log"

	"github.com/paveltessman/yaa/platform/api/callbacks"
	"github.com/paveltessman/yaa/platform/telegram"
)

type lifespan func(Deps) error

var _ lifespan = tearUp
var _ lifespan = tearDown

func tearUp(deps Deps) error {
	params := telegram.SetWebhookParams{
		URL:            deps.settings.PublicHost + callbacks.TgWebhookPath,
		AllowedUpdates: []string{"message"},
	}
	err := deps.tgClient.SetWebhook(params)
	if err != nil {
		return err
	}
	log.Println("Tear up done!")
	return nil
}

func tearDown(deps Deps) error {
	err := deps.tgClient.DeleteWebhook()
	if err != nil {
		return err
	}
	log.Println("Tear down done!")
	return nil
}
