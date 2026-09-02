package models

import "context"

var _ DBRepo = (*FakeDBRepo)(nil)

// FakeDBRepo records every call and returns Error. The zero value is ready to
// use.
type FakeDBRepo struct {
	Error      error
	CallCounts map[string]int
	Messages   []*Message
	Contexts   []context.Context
}

func (r *FakeDBRepo) StoreMessage(ctx context.Context, message *Message) error {
	if r.CallCounts == nil {
		r.CallCounts = make(map[string]int)
	}
	r.CallCounts["StoreMessage"]++
	r.Messages = append(r.Messages, message)
	r.Contexts = append(r.Contexts, ctx)
	return r.Error
}
