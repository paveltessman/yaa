package handlers

import (
	"context"
	"errors"
	"fmt"

	"github.com/paveltessman/yaa/pipelines/telegram/ports"
	"github.com/paveltessman/yaa/pipelines/telegram/updates/session"
)

var ErrStoreMessage = errors.New("unable to store message")

// pickMessage takes the message that the handler stores from the session.
type pickMessage func(*session.Session) *ports.Message

type StoreMessage struct {
	repo ports.DBRepo
	pick pickMessage
}

func newStoreMessage(repo ports.DBRepo, pick pickMessage) StoreMessage {
	if repo == nil {
		panic("db repo object is nil")
	}

	h := StoreMessage{repo: repo, pick: pick}
	return h
}

// NewStoreMessage stores the message of the update (the message from user).
func NewStoreMessage(repo ports.DBRepo) StoreMessage {
	pick := func(session *session.Session) *ports.Message { return session.Message }
	return newStoreMessage(repo, pick)
}

// NewStoreReply stores the reply from agent.
func NewStoreReply(repo ports.DBRepo) StoreMessage {
	pick := func(session *session.Session) *ports.Message { return session.SentMessage }
	return newStoreMessage(repo, pick)
}

func (h StoreMessage) Handle(ctx context.Context, session *session.Session) error {
	message := h.pick(session)
	if message == nil {
		return fmt.Errorf("%w: session has no message", ErrStoreMessage)
	}

	err := h.repo.StoreMessage(ctx, message)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrStoreMessage, err)
	}
	return nil
}
