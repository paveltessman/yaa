package background

import (
	"context"
	"errors"
	"io"
	"log"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

const testTimeout = 5 * time.Second

func quietLog(t *testing.T) {
	t.Helper()
	log.SetOutput(io.Discard)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })
}

func waitOK(t *testing.T, runner *Runner) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	if err := runner.Wait(ctx); err != nil {
		t.Fatalf("want the tasks to finish, got %v", err)
	}
}

func TestGoRunsTheTask(t *testing.T) {
	runner := NewRunner(2, testTimeout)
	done := make(chan struct{})

	if !runner.Go(context.Background(), func(context.Context) { close(done) }) {
		t.Fatal("want a started task, got a refusal")
	}
	waitOK(t, runner)

	select {
	case <-done:
	default:
		t.Error("want the task to run, got no call")
	}
}

func TestWaitBlocksUntilTheTaskEnds(t *testing.T) {
	runner := NewRunner(2, testTimeout)
	var ended atomic.Bool

	runner.Go(context.Background(), func(context.Context) {
		time.Sleep(50 * time.Millisecond)
		ended.Store(true)
	})
	waitOK(t, runner)

	if !ended.Load() {
		t.Error("want Wait to block until the task ends, got an early return")
	}
}

func TestTaskContextOutlivesTheRequest(t *testing.T) {
	type key struct{}
	runner := NewRunner(2, testTimeout)
	parent, cancel := context.WithCancel(context.WithValue(context.Background(), key{}, "marker"))

	values := make(chan any, 1)
	errs := make(chan error, 1)
	started := make(chan struct{})
	release := make(chan struct{})

	runner.Go(parent, func(ctx context.Context) {
		close(started)
		<-release
		values <- ctx.Value(key{})
		errs <- ctx.Err()
	})

	<-started
	cancel() // the server cancels the request context after the handler
	close(release)
	waitOK(t, runner)

	if got := <-values; got != "marker" {
		t.Errorf("want=%q, got=%v", "marker", got)
	}
	if err := <-errs; err != nil {
		t.Errorf("want a live context, got %v", err)
	}
}

func TestTaskContextHasTheTimeout(t *testing.T) {
	runner := NewRunner(2, 100*time.Millisecond)
	errs := make(chan error, 1)

	runner.Go(context.Background(), func(ctx context.Context) {
		<-ctx.Done()
		errs <- ctx.Err()
	})
	waitOK(t, runner)

	if err := <-errs; !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("want a deadline error, got %v", err)
	}
}

func TestGoRefusesAfterStop(t *testing.T) {
	runner := NewRunner(2, testTimeout)
	runner.Stop()

	var ran atomic.Bool
	if runner.Go(context.Background(), func(context.Context) { ran.Store(true) }) {
		t.Error("want a refusal after Stop, got a started task")
	}
	waitOK(t, runner)

	if ran.Load() {
		t.Error("want no call after Stop, got one")
	}
}

func TestPanicDoesNotStopTheRunner(t *testing.T) {
	quietLog(t)
	runner := NewRunner(2, testTimeout)
	var ran atomic.Bool

	runner.Go(context.Background(), func(context.Context) { panic("task failed") })
	runner.Go(context.Background(), func(context.Context) { ran.Store(true) })
	waitOK(t, runner)

	if !ran.Load() {
		t.Error("want the second task to run, got no call")
	}
}

func TestLimitBoundsTheRunningTasks(t *testing.T) {
	const limit = 2
	const tasks = 6
	runner := NewRunner(limit, testTimeout)

	var mu sync.Mutex
	var running, peak int

	for range tasks {
		runner.Go(context.Background(), func(context.Context) {
			mu.Lock()
			running++
			peak = max(peak, running)
			mu.Unlock()

			time.Sleep(20 * time.Millisecond)

			mu.Lock()
			running--
			mu.Unlock()
		})
	}
	waitOK(t, runner)

	mu.Lock()
	defer mu.Unlock()
	if peak > limit {
		t.Errorf("want at most %d running tasks, got %d", limit, peak)
	}
}

func TestWaitReturnsTheContextError(t *testing.T) {
	quietLog(t)
	runner := NewRunner(1, 500*time.Millisecond)
	runner.Go(context.Background(), func(ctx context.Context) { <-ctx.Done() })

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	if err := runner.Wait(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("want a deadline error, got %v", err)
	}
}

func TestStopKeepsTheAcceptedTasks(t *testing.T) {
	runner := NewRunner(1, testTimeout)

	release := make(chan struct{})
	holds := make(chan struct{})
	var second atomic.Bool

	runner.Go(context.Background(), func(context.Context) {
		close(holds)
		<-release
	})
	<-holds
	if !runner.Go(context.Background(), func(context.Context) { second.Store(true) }) {
		t.Fatal("want a started task, got a refusal")
	}

	runner.Stop()
	close(release)
	waitOK(t, runner)

	if !second.Load() {
		t.Error("want the accepted task to run after Stop, got no call")
	}
}

func TestNewRunnerRejectsBadArguments(t *testing.T) {
	cases := map[string]struct {
		limit   int
		timeout time.Duration
	}{
		"zero limit":       {0, testTimeout},
		"negative limit":   {-1, testTimeout},
		"zero timeout":     {1, 0},
		"negative timeout": {1, -time.Second},
	}
	for name, key := range cases {
		t.Run(name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Error("want a panic, got none")
				}
			}()
			NewRunner(key.limit, key.timeout)
		})
	}
}
