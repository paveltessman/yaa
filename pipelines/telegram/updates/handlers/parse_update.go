package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"

	"github.com/paveltessman/yaa/pipelines/telegram/updates/session"
	"github.com/paveltessman/yaa/platform/telegram"
)

var ErrParseUpdate = errors.New("can't parse telegram update")

func ParseUpdate(ctx context.Context, session *session.Session) error {
	if len(session.RawUpdate) == 0 {
		return fmt.Errorf("%w: update is empty", ErrParseUpdate)
	}
	var update telegram.Update

	err := json.Unmarshal(session.RawUpdate, &update)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrParseUpdate, err)
	}

	if update.Message == nil {
		log.Println("telegram update discarded: no message")
		return nil
	}
	session.Update = &update
	log.Println(update.Message)
	return nil
}
