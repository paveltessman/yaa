package history

import (
	"context"

	"github.com/paveltessman/yaa/pipelines/shared/ports/history"
)

var _ history.HistoryService = (*FakeHistoryService)(nil)

type FakeHistoryService struct {
	Error error

	Calls    int
	Contexts []context.Context
	Records  []*history.Record
}

func (f *FakeHistoryService) Save(ctx context.Context, record *history.Record) error {
	f.Calls++
	f.Contexts = append(f.Contexts, ctx)
	f.Records = append(f.Records, record)
	return f.Error
}
