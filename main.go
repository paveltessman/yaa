package main

import (
	"log"

	"github.com/paveltessman/yaa/platform/network"
	"github.com/paveltessman/yaa/platform/settings"
)

func main() {
	log.Println("Started!")
	_ = settings.NewSettings()
	log.Println("Settings loaded")

	session := network.NewHTTPSession("https://api.ipify.org")

	ip, err := session.GetBytes("")
	if err != nil {
		panic(err)
	}
	log.Printf("ip: %s", string(ip))
}
