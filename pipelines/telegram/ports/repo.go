package ports

import (
	"context"
	"time"
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

type DBRepo interface {
	StoreMessage(context.Context, *Message) error
	LoadThread(ctx context.Context, chatID, threadID int64) ([]*Message, error)
}
