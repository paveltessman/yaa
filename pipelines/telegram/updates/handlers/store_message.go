package handlers

import (
	"context"
	"errors"
	"fmt"

	"github.com/paveltessman/yaa/pipelines/telegram/updates/models"
	"github.com/paveltessman/yaa/pipelines/telegram/updates/session"
)

var ErrStoreMessage = errors.New("unable to store message")

type StoreMessage struct {
	repo models.DBRepo
}

func NewStoreMessage(repo models.DBRepo) StoreMessage {
	if repo == nil {
		panic("db repo object is nil")
	}

	h := StoreMessage{repo: repo}
	return h
}

func (h StoreMessage) Handle(ctx context.Context, session *session.Session) error {
	message := session.Message
	if message == nil {
		return fmt.Errorf("%w: session has no message", ErrStoreMessage)
	}

	err := h.repo.StoreMessage(ctx, message)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrStoreMessage, err)
	}
	return nil
}
