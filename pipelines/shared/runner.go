package shared

import (
	"context"
	"errors"
)

var ErrCompleted = errors.New("pass completed")

type Runner[S Session] func(context.Context, S, []Handler[S]) error

func Run[S Session](ctx context.Context, session S, chain []Handler[S]) error {
	for _, handler := range chain {
		err := handler.Handle(ctx, session)
		if errors.Is(err, ErrCompleted) {
			return nil
		}
		if err != nil {
			return err
		}
	}
	return nil
}
