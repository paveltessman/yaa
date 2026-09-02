package models

import (
	"testing"

	"github.com/paveltessman/yaa/platform/telegram"
)

func newTgMessage() telegram.Message {
	tgMessage := telegram.Message{
		ID:       1,
		ThreadID: 2,
		From: telegram.User{
			ID: 888,
		},
		Chat: telegram.Chat{
			ID: 777,
		},
		Text: "yo bro",
		Date: 1788382339,
	}
	return tgMessage
}

func TestFromTgMessage(t *testing.T) {
	tgMessage := newTgMessage()

	message := FromTgMessage(&tgMessage)

	if got, want := message.ID, tgMessage.ID; got != want {
		t.Errorf("ids don't match: our=%d, tg=%d", got, want)
	}
	if got, want := message.ChatID, tgMessage.Chat.ID; got != want {
		t.Errorf("chat ids don't match: our=%d, tg=%d", got, want)
	}
	if got, want := message.ThreadID, tgMessage.ThreadID; got != want {
		t.Errorf("thread ids don't match: our=%d, tg=%d", got, want)
	}
	if got, want := message.UserID, tgMessage.From.ID; got != want {
		t.Errorf("user ids don't match: our=%d, tg=%d", got, want)
	}
	if got, want := message.Date, tgMessage.Time(); got != want {
		t.Errorf("dates don't match: our=%s, tg=%s", got, want)
	}
	if got, want := message.Type, FromUser; got != want {
		t.Errorf("message type is incorrect: got=%s, want=%s", got, want)
	}
	if got, want := message.Text, tgMessage.Text; got != want {
		t.Errorf("text doesn't match: our=%s, tg=%s", got, want)
	}
}
