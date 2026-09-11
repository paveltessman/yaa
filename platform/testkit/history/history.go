package history

import (
	"context"
	"fmt"
	"uuid"

	"github.com/paveltessman/yaa/pipelines/shared/ports/history"
)

var _ history.HistoryService = (*FakeHistoryService)(nil)

type FakeHistoryService struct {
	Error error

	Calls    int
	Contexts []context.Context
	Records  []*history.Record

	// Stored answers List and Get.
	Stored []*history.Record

	ListCalls  int
	ListLimits []int32
	GetCalls   int
	GetIDs     []uuid.UUID
}

func (f *FakeHistoryService) Save(ctx context.Context, record *history.Record) error {
	f.Calls++
	f.Contexts = append(f.Contexts, ctx)
	f.Records = append(f.Records, record)
	return f.Error
}

func (f *FakeHistoryService) List(ctx context.Context, limit int32) ([]*history.Record, error) {
	f.ListCalls++
	f.Contexts = append(f.Contexts, ctx)
	f.ListLimits = append(f.ListLimits, limit)
	if f.Error != nil {
		return nil, f.Error
	}

	records := make([]*history.Record, 0, len(f.Stored))
	for _, record := range f.Stored {
		if int32(len(records)) == limit {
			break
		}
		records = append(records, record)
	}
	return records, nil
}

func (f *FakeHistoryService) Get(ctx context.Context, sessionID uuid.UUID) (*history.Record, error) {
	f.GetCalls++
	f.Contexts = append(f.Contexts, ctx)
	f.GetIDs = append(f.GetIDs, sessionID)
	if f.Error != nil {
		return nil, f.Error
	}

	for _, record := range f.Stored {
		if record.SessionID == sessionID {
			return record, nil
		}
	}
	return nil, fmt.Errorf("session %s: %w", sessionID, history.ErrNotFound)
}
