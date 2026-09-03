package models

import "context"

type DBRepo interface {
	StoreMessage(context.Context, *Message) error
	LoadThread(ctx context.Context, chatID, threadID int64) ([]*Message, error)
}
