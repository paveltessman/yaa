package main

import (
	"log"

	"github.com/paveltessman/yaa/platform/settings"
	"github.com/paveltessman/yaa/platform/telegram"
)

func main() {
	log.Println("Started!")
	s := settings.NewSettings()
	log.Println("Settings loaded")

	tgClient := telegram.NewClient(telegram.NewSession(s.TgToken))
	me, err := tgClient.GetMe()
	if err != nil {
		panic(err)
	}
	log.Println(me)
}
