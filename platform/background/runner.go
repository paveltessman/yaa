// Package background runs work after the handler answered the client.
package background

import (
	"context"
	"log"
	"runtime/debug"
	"sync"
	"time"
)

type Runner struct {
	slots   chan struct{}
	timeout time.Duration
	wg      sync.WaitGroup
	mu      sync.Mutex
	stopped bool
}

// NewRunner makes a runner. The limit is the number of tasks that run at
// the same time. The timeout is the time budget of one task.
func NewRunner(limit int, timeout time.Duration) *Runner {
	if limit <= 0 {
		panic("limit must be positive")
	}
	if timeout <= 0 {
		panic("timeout must be positive")
	}
	r := &Runner{
		slots:   make(chan struct{}, limit),
		timeout: timeout,
	}
	return r
}

func (r *Runner) Go(parent context.Context, fn func(context.Context)) bool {
	if !r.add() {
		return false
	}
	ctx := context.WithoutCancel(parent)

	go func() {
		defer r.wg.Done()
		defer logPanic()

		r.slots <- struct{}{}
		defer func() { <-r.slots }()

		ctx, cancel := context.WithTimeout(ctx, r.timeout)
		defer cancel()

		fn(ctx)
	}()
	return true
}

// Stop rejects new tasks. A task that Go accepted still runs.
func (r *Runner) Stop() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.stopped = true
}

// Wait stops the runner and blocks until the running tasks finish.
func (r *Runner) Wait(ctx context.Context) error {
	r.Stop()

	done := make(chan struct{})
	go func() {
		r.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (r *Runner) add() bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.stopped {
		return false
	}
	r.wg.Add(1)
	return true
}

func logPanic() {
	value := recover()
	if value == nil {
		return
	}
	log.Printf("background task panicked: %v\n%s", value, debug.Stack())
}
