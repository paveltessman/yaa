package history

import (
	"context"
	"uuid"

	"github.com/paveltessman/yaa/pipelines/shared/ports/history"
)

var _ history.HistoryService = (*FakeHistoryService)(nil)

type FakeHistoryService struct {
}

func (f *FakeHistoryService) Save(ctx context.Context, sessionID uuid.UUID, entries []history.Entry) error {
	return nil
}
