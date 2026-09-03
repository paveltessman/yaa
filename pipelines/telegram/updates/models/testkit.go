package models

import "context"

var _ DBRepo = (*FakeDBRepo)(nil)

// LoadThreadCall holds the arguments of one LoadThread call.
type LoadThreadCall struct {
	ChatID   int64
	ThreadID int64
}

type FakeDBRepo struct {
	Error           error
	CallCounts      map[string]int
	Messages        []*Message
	Contexts        []context.Context
	Thread          []*Message
	LoadThreadCalls []LoadThreadCall
}

func (r *FakeDBRepo) StoreMessage(ctx context.Context, message *Message) error {
	r.record(ctx, "StoreMessage")
	r.Messages = append(r.Messages, message)
	return r.Error
}

func (r *FakeDBRepo) LoadThread(ctx context.Context, chatID, threadID int64) ([]*Message, error) {
	r.record(ctx, "LoadThread")
	r.LoadThreadCalls = append(r.LoadThreadCalls, LoadThreadCall{ChatID: chatID, ThreadID: threadID})
	if r.Error != nil {
		return nil, r.Error
	}
	return r.Thread, nil
}

func (r *FakeDBRepo) record(ctx context.Context, name string) {
	if r.CallCounts == nil {
		r.CallCounts = make(map[string]int)
	}
	r.CallCounts[name]++
	r.Contexts = append(r.Contexts, ctx)
}
