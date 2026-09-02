package shared

import "context"

type Handler[S Session] interface {
	Handle(context.Context, S) error
}

type HandlerFunc[S Session] func(context.Context, S) error

func (h HandlerFunc[S]) Handle(ctx context.Context, session S) error {
	return h(ctx, session)
}
