package api

import (
	"context"
	"errors"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/paveltessman/yaa/pipelines/agent"
	"github.com/paveltessman/yaa/pipelines/telegram/updates"
	"github.com/paveltessman/yaa/platform/api/callbacks"
	"github.com/paveltessman/yaa/platform/background"
)

const shutdownTimeout = 10 * time.Second
const tearDownTimeout = 10 * time.Second
const readHeaderTimeout = 10 * time.Second
const readTimeout = 15 * time.Second
const writeTimeout = 15 * time.Second
const idleTimeout = 60 * time.Second

const maxBackgroundTasks = 64
const backgroundTaskTimeout = 5 * time.Minute
const backgroundWaitTimeout = 30 * time.Second

func NewRouter(deps Deps, runner *background.Runner) http.Handler {
	mux := http.NewServeMux()

	agentPipeline := agent.NewPipeline(deps.llmService, deps.historyService)
	tgUpdatesPipeline := updates.NewPipeline(deps.tgClient, deps.dbRepo, agentPipeline, deps.historyService)

	mux.Handle(callbacks.TgWebhookPath, callbacks.Telegram(tgUpdatesPipeline, runner))

	return mux
}

func Serve(ctx context.Context, deps Deps) error {
	runner := background.NewRunner(maxBackgroundTasks, backgroundTaskTimeout)
	return serve(ctx, deps, NewRouter(deps, runner), runner, tearUp, tearDown)
}

func serve(
	ctx context.Context,
	deps Deps,
	handler http.Handler,
	runner *background.Runner,
	tearUp, tearDown lifespan,
) (err error) {
	if err := tearUp(ctx, deps); err != nil {
		return err
	}
	defer func() {
		tearDownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), tearDownTimeout)
		defer cancel()

		if tearDownErr := tearDown(tearDownCtx, deps); tearDownErr != nil {
			log.Printf("tear down failed: %v", tearDownErr)
			if err == nil {
				err = tearDownErr
			}
		}
	}()

	defer waitBackground(ctx, runner)

	listener, err := net.Listen("tcp", deps.settings.ApiAddr)
	if err != nil {
		return err
	}

	srv := &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
	}

	errs := make(chan error, 1)

	go func() {
		errs <- srv.Serve(listener)
	}()

	log.Printf("Serving on %s", listener.Addr())

	select {
	case err := <-errs:
		return ignoreServerClosed(err)
	case <-ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), shutdownTimeout)
	defer cancel()
	if shutdownErr := srv.Shutdown(shutdownCtx); shutdownErr != nil {
		return errors.Join(shutdownErr, ignoreServerClosed(<-errs))
	}
	return ignoreServerClosed(<-errs)
}

func waitBackground(ctx context.Context, runner *background.Runner) {
	waitCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), backgroundWaitTimeout)
	defer cancel()

	if err := runner.Wait(waitCtx); err != nil {
		log.Printf("background tasks did not finish: %v", err)
	}
}

func ignoreServerClosed(err error) error {
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}
