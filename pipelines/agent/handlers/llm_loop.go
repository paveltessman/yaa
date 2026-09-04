package handlers

import (
	"context"

	"github.com/paveltessman/yaa/pipelines/agent/session"
)

func LlmLoop(ctx context.Context, session *session.Session) error {
	session.Reply = "helow!"
	return nil
}
