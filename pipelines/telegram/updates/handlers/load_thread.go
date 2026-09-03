package handlers

import (
	"context"
	"errors"
	"fmt"

	"github.com/paveltessman/yaa/pipelines/telegram/updates/models"
	"github.com/paveltessman/yaa/pipelines/telegram/updates/session"
)

var ErrLoadThread = errors.New("unable to load thread")

type LoadThread struct {
	repo models.DBRepo
}

func NewLoadThread(repo models.DBRepo) LoadThread {
	if repo == nil {
		panic("db repo object is nil")
	}

	h := LoadThread{repo: repo}
	return h
}

func (h LoadThread) Handle(ctx context.Context, session *session.Session) error {
	message := session.Message
	if message == nil {
		return fmt.Errorf("%w: session has no message", ErrLoadThread)
	}

	thread, err := h.repo.LoadThread(ctx, message.ChatID, message.ThreadID)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrLoadThread, err)
	}

	session.Thread = thread
	return nil
}
