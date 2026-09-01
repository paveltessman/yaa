package main

import (
	"context"
	"log"

	"github.com/paveltessman/yaa/platform/api"
	"github.com/paveltessman/yaa/platform/settings"
	"github.com/paveltessman/yaa/platform/telegram"
)

func main() {
	s := settings.NewSettings()
	log.Println("Settings loaded")

	tgClient := telegram.NewClient(telegram.NewSession(s.TgToken))
	me, err := tgClient.GetMe()
	if err != nil {
		panic(err)
	}
	log.Println(me)

	deps := api.NewDeps(&s, tgClient)

	if err := api.Serve(context.Background(), deps); err != nil {
		panic(err)
	}
}
