package main

import (
	"log"

	"github.com/paveltessman/yaa/settings"
)

func main() {
	log.Println("Started!")
	_ = settings.NewSettings()
	log.Println("Settings loaded")
}
