package telegram

import (
	"context"

	"github.com/paveltessman/yaa/pipelines/telegram/ports"
)

var _ ports.DBRepo = (*FakeDBRepo)(nil)

// LoadThreadCall holds the arguments of one LoadThread call.
type LoadThreadCall struct {
	ChatID   int64
	ThreadID int64
}

type FakeDBRepo struct {
	Error           error
	CallCounts      map[string]int
	Messages        []*ports.Message
	Contexts        []context.Context
	Thread          []*ports.Message
	LoadThreadCalls []LoadThreadCall
}

func (r *FakeDBRepo) StoreMessage(ctx context.Context, message *ports.Message) error {
	r.record(ctx, "StoreMessage")
	r.Messages = append(r.Messages, message)
	return r.Error
}

func (r *FakeDBRepo) LoadThread(ctx context.Context, chatID, threadID int64) ([]*ports.Message, error) {
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
