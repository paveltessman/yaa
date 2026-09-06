package api

import (
	"context"
	"log"

	"github.com/paveltessman/yaa/pipelines/telegram/ports"
	"github.com/paveltessman/yaa/platform/api/callbacks"
)

type lifespan func(context.Context, Deps) error

var _ lifespan = tearUp
var _ lifespan = tearDown

func tearUp(ctx context.Context, deps Deps) error {
	params := ports.SetWebhookParams{
		URL:            deps.settings.PublicHost + callbacks.TgWebhookPath,
		AllowedUpdates: []string{"message"},
	}
	err := deps.tgClient.SetWebhook(ctx, params)
	if err != nil {
		return err
	}
	log.Println("Tear up done!")
	return nil
}

func tearDown(ctx context.Context, deps Deps) error {
	err := deps.tgClient.DeleteWebhook(ctx)
	if err != nil {
		return err
	}
	log.Println("Tear down done!")
	return nil
}
