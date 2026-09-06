package handlers

import (
	"context"
	"errors"
	"fmt"

	"github.com/paveltessman/yaa/pipelines/telegram/ports"
	"github.com/paveltessman/yaa/pipelines/telegram/updates/session"
)

var ErrSendReply = errors.New("unable to send reply")

type SendReply struct {
	client ports.Sender
}

func NewSendReply(client ports.Sender) SendReply {
	if client == nil {
		panic("tg client object is nil")
	}

	h := SendReply{client: client}
	return h
}

func (h SendReply) Handle(ctx context.Context, session *session.Session) error {
	message := session.Message
	if message == nil {
		return fmt.Errorf("%w: session has no message", ErrSendReply)
	}
	if session.Reply == "" {
		return fmt.Errorf("%w: session has no reply", ErrSendReply)
	}

	params := ports.SendMessageParams{
		ChatID:   message.ChatID,
		ThreadID: message.ThreadID,
		Text:     session.Reply,
	}

	sent, err := h.client.SendMessage(ctx, params)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrSendReply, err)
	}

	session.SentMessage = sent
	return nil
}
