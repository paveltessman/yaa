package models

import "context"

type DBRepo interface {
	StoreMessage(context.Context, *Message) error
}
