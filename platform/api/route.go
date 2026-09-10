package api

import (
	"context"
	"errors"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/paveltessman/yaa/pipelines/agent"
	"github.com/paveltessman/yaa/pipelines/shared/ports/history"
	"github.com/paveltessman/yaa/pipelines/telegram/updates"
	"github.com/paveltessman/yaa/platform/api/callbacks"
	testkit "github.com/paveltessman/yaa/platform/testkit/history"
)

const shutdownTimeout = 10 * time.Second
const tearDownTimeout = 10 * time.Second
const readHeaderTimeout = 10 * time.Second
const readTimeout = 15 * time.Second
const writeTimeout = 15 * time.Second
const idleTimeout = 60 * time.Second

func fakeHistoryService() history.HistoryService {
	return &testkit.FakeHistoryService{}
}

func NewRouter(deps Deps) http.Handler {
	mux := http.NewServeMux()

	agentPipeline := agent.NewPipeline(deps.llmService, fakeHistoryService())
	tgUpdatesPipeline := updates.NewPipeline(deps.tgClient, deps.dbRepo, agentPipeline, fakeHistoryService())

	mux.Handle(callbacks.TgWebhookPath, callbacks.Telegram(tgUpdatesPipeline))

	return mux
}

func Serve(ctx context.Context, deps Deps) error {
	return serve(ctx, deps, NewRouter(deps), tearUp, tearDown)
}

func serve(ctx context.Context, deps Deps, handler http.Handler, tearUp, tearDown lifespan) (err error) {
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

func ignoreServerClosed(err error) error {
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}
