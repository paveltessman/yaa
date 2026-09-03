package main

import (
	"context"
	"log"

	"github.com/paveltessman/yaa/platform/api"
	"github.com/paveltessman/yaa/platform/db"
	"github.com/paveltessman/yaa/platform/db/repos/tgupdates"
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

	pool, err := db.NewPool(ctx, s.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()
	log.Println("Database pool ready")

	deps := api.NewDeps(&s, tgClient, tgupdates.New(pool))

	if err := api.Serve(ctx, deps); err != nil {
		return err
	}
	return nil
}
