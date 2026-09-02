package handlers

import (
	"context"
	"log"

	"github.com/paveltessman/yaa/pipelines/telegram/updates/session"
)

func ParseUpdate(ctx context.Context, session *session.Session) error {
	log.Printf("update data: %s", session.RawUpdate)
	log.Printf("session id: %s", session.ID())
	log.Printf("session date: %s", session.Date())
	log.Println("Works!!!")
	return nil
}
