package api

import (
	"context"
	"errors"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/paveltessman/yaa/platform/api/callbacks"
)

const shutdownTimeout = 10 * time.Second
const readHeaderTimeout = 10 * time.Second
const readTimeout = 15 * time.Second
const writeTimeout = 15 * time.Second
const idleTimeout = 60 * time.Second

func NewRouter() http.Handler {
	mux := http.NewServeMux()

	mux.Handle("/v1/callbacks/telegram", callbacks.Telegram())
	return mux
}

func tearUp() error {
	log.Println("Tear up done!")
	return nil
}

func tearDown() error {
	log.Println("Tear down done!")
	return nil
}

func Serve(ctx context.Context, addr string) error {
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()
	return serve(ctx, addr, NewRouter(), tearUp, tearDown)
}

func serve(ctx context.Context, addr string, handler http.Handler, tearUp, tearDown func() error) (err error) {
	if err := tearUp(); err != nil {
		return err
	}
	defer func() {
		if tearDownErr := tearDown(); tearDownErr != nil {
			log.Printf("tear down failed: %v", tearDownErr)
			if err == nil {
				err = tearDownErr
			}
		}
	}()

	listener, err := net.Listen("tcp", addr)
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
