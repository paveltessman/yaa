package api

import (
	"context"
	"errors"
	"log"
	"net"
	"net/http"
	"time"

	pipelines "github.com/paveltessman/yaa/pipelines/shared"
	"github.com/paveltessman/yaa/pipelines/telegram/updates"
	"github.com/paveltessman/yaa/platform/api/callbacks"
)

const shutdownTimeout = 10 * time.Second
const readHeaderTimeout = 10 * time.Second
const readTimeout = 15 * time.Second
const writeTimeout = 15 * time.Second
const idleTimeout = 60 * time.Second

func NewRouter(deps Deps) http.Handler {
	mux := http.NewServeMux()

	chain := updates.NewChain(deps.dbRepo)
	mux.Handle(callbacks.TgWebhookPath, callbacks.Telegram(pipelines.Run, chain))
	return mux
}

func Serve(ctx context.Context, deps Deps) error {
	return serve(ctx, deps, NewRouter(deps), tearUp, tearDown)
}

func serve(ctx context.Context, deps Deps, handler http.Handler, tearUp, tearDown lifespan) (err error) {
	if err := tearUp(deps); err != nil {
		return err
	}
	defer func() {
		if tearDownErr := tearDown(deps); tearDownErr != nil {
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
