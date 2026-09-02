package models

import (
	"time"

	"github.com/paveltessman/yaa/platform/telegram"
)

type MessageType string

const (
	FromUser MessageType = "from_user"
	ToUser   MessageType = "to_user"
)

type Message struct {
	ID       int64
	ChatID   int64
	ThreadID int64
	UserID   int64
	Type     MessageType
	Date     time.Time
	Text     string
}

func FromTgMessage(m *telegram.Message) *Message {
	message := Message{
		ID:       m.ID,
		ChatID:   m.Chat.ID,
		ThreadID: m.ThreadID,
		UserID:   m.From.ID,
		Type:     FromUser,
		Date:     m.Time(),
		Text:     m.Text,
	}
	return &message
}
