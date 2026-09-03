package shared

import (
	"context"
	"errors"
)

var ErrCompleted = errors.New("pass completed")

type ErrorHandler[S Session] func(context.Context, S, error) error

type Chain[S Session] struct {
	chain        []Handler[S]
	errorHandler ErrorHandler[S]
}

func NewChain[S Session](chain []Handler[S], errorHandler ErrorHandler[S]) Chain[S] {
	c := Chain[S]{
		chain:        chain,
		errorHandler: errorHandler,
	}
	return c
}

type Runner[S Session] func(context.Context, S, Chain[S]) error

func Run[S Session](ctx context.Context, session S, chain Chain[S]) error {
	for _, handler := range chain.chain {
		err := handler.Handle(ctx, session)
		if errors.Is(err, ErrCompleted) {
			return nil
		}
		if err == nil {
			continue
		}
		if handledErr := chain.errorHandler(ctx, session, err); handledErr != nil {
			return handledErr
		}
		return err
	}
	return nil
}
