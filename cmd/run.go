package main

import (
	"context"
	"log"

	"github.com/paveltessman/yaa/platform/api"
	"github.com/paveltessman/yaa/platform/settings"
	"github.com/paveltessman/yaa/platform/telegram"
)

func run(ctx context.Context, _ []string) error {
	s := settings.NewSettings()
	log.Println("Settings loaded")

	tgClient := telegram.NewClient(telegram.NewSession(s.TgToken))
	_, err := tgClient.GetMe()
	if err != nil {
		return err
	}

	deps := api.NewDeps(&s, tgClient)

	if err := api.Serve(ctx, deps); err != nil {
		return err
	}
	return nil
}
