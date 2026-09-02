package shared

import "context"

type Runner[S Session] func(context.Context, S, []Handler[S]) error

func Run[S Session](ctx context.Context, session S, chain []Handler[S]) error {
	for _, handler := range chain {
		if err := handler.Handle(ctx, session); err != nil {
			return err
		}
	}
	return nil
}
