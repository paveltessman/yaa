package api

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/paveltessman/yaa/pipelines/telegram/updates/models"
	"github.com/paveltessman/yaa/platform/api/callbacks"
	"github.com/paveltessman/yaa/platform/settings"
	"github.com/paveltessman/yaa/platform/telegram"
)

var errTearUp = errors.New("tear up failed")
var errTearDown = errors.New("tear down failed")

func noopHook(deps Deps) error { return nil }

func defaultDeps() Deps {
	s := settings.Settings{
		TgToken:    "12345:abcdefg",
		PublicHost: "http://example.com",
		ApiAddr:    "127.0.0.1:8080",
	}
	deps := Deps{
		settings: &s,
		tgClient: &telegram.FakeClient{},
		dbRepo:   &models.FakeDBRepo{},
	}
	return deps

}

func withAddr(deps Deps, addr string) Deps {
	deps.settings.ApiAddr = addr
	return deps
}

func failingHook(err error) func(Deps) error {
	return func(deps Deps) error { return err }
}

func countingHook(calls *int, err error) func(Deps) error {
	return func(deps Deps) error {
		*calls++
		return err
	}
}

func freeAddr(t *testing.T) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("want a free port, got %v", err)
	}
	addr := listener.Addr().String()
	if err := listener.Close(); err != nil {
		t.Fatalf("want a closed listener, got %v", err)
	}
	return addr
}

func busyAddr(t *testing.T) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("want a free port, got %v", err)
	}
	t.Cleanup(func() { _ = listener.Close() })
	return listener.Addr().String()
}

func startServe(t *testing.T, handler http.Handler, deps Deps, tearUp, tearDown lifespan) (string, context.CancelFunc, <-chan error) {
	t.Helper()
	addr := freeAddr(t)
	deps = withAddr(deps, addr)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	errs := make(chan error, 1)
	go func() {
		errs <- serve(ctx, deps, handler, tearUp, tearDown)
	}()
	return addr, cancel, errs
}

func waitForServer(t *testing.T, addr string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, 100*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("want a server on %s, got no answer", addr)
}

func waitResult(t *testing.T, errs <-chan error) error {
	t.Helper()
	select {
	case err := <-errs:
		return err
	case <-time.After(5 * time.Second):
		t.Fatal("want serve to return, got a timeout")
		return nil
	}
}

func TestServeCleanShutdownReturnsNil(t *testing.T) {
	tearDownCalls := 0
	deps := defaultDeps()
	addr, cancel, errs := startServe(t, NewRouter(deps), deps, noopHook, countingHook(&tearDownCalls, nil))
	waitForServer(t, addr)

	cancel()

	if err := waitResult(t, errs); err != nil {
		t.Errorf("want no error on a clean shutdown, got %v", err)
	}
	if tearDownCalls != 1 {
		t.Errorf("want 1 tear down call, got %d", tearDownCalls)
	}
}

func TestServeRoutesRequests(t *testing.T) {
	deps := defaultDeps()
	addr, cancel, errs := startServe(t, NewRouter(deps), deps, noopHook, noopHook)
	waitForServer(t, addr)

	resp, err := http.Post("http://"+addr+"/v1/callbacks/telegram", "application/json", strings.NewReader(`{"update_id":1}`))
	if err != nil {
		t.Fatalf("want a response, got %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("want=%d, got=%d", http.StatusOK, resp.StatusCode)
	}

	cancel()
	if err := waitResult(t, errs); err != nil {
		t.Errorf("want no error on a clean shutdown, got %v", err)
	}
}

func TestServeWaitsForInFlightRequest(t *testing.T) {
	var once sync.Once
	started := make(chan struct{})
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		once.Do(func() { close(started) })
		time.Sleep(200 * time.Millisecond)
		_, _ = w.Write([]byte("done"))
	})

	addr, cancel, errs := startServe(t, handler, defaultDeps(), noopHook, noopHook)
	waitForServer(t, addr)

	bodies := make(chan string, 1)
	go func() {
		resp, err := http.Get("http://" + addr + "/")
		if err != nil {
			bodies <- "request failed: " + err.Error()
			return
		}
		defer func() {
			_ = resp.Body.Close()
		}()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			bodies <- "read failed: " + err.Error()
			return
		}
		bodies <- string(body)
	}()

	<-started
	cancel()

	select {
	case got := <-bodies:
		if got != "done" {
			t.Errorf("want=%q, got=%q", "done", got)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("want the in flight request to finish, got a timeout")
	}

	if err := waitResult(t, errs); err != nil {
		t.Errorf("want no error on a clean shutdown, got %v", err)
	}
}

func TestServeListenErrorSurfaces(t *testing.T) {
	tearDownCalls := 0

	deps := withAddr(defaultDeps(), busyAddr(t))
	err := serve(context.Background(), deps, NewRouter(deps), noopHook, countingHook(&tearDownCalls, nil))

	if err == nil {
		t.Error("want a listen error on a busy address, got nil")
	}
	if tearDownCalls != 1 {
		t.Errorf("want 1 tear down call after a listen error, got %d", tearDownCalls)
	}
}

func TestServeTearUpErrorStopsTheServer(t *testing.T) {
	tearDownCalls := 0

	deps := withAddr(defaultDeps(), freeAddr(t))
	err := serve(context.Background(), deps, NewRouter(deps), failingHook(errTearUp), countingHook(&tearDownCalls, nil))

	if !errors.Is(err, errTearUp) {
		t.Errorf("want errTearUp, got %v", err)
	}
	if tearDownCalls != 0 {
		t.Errorf("want no tear down after a tear up failure, got %d calls", tearDownCalls)
	}
}

func TestServeTearDownErrorSurfacesAfterACleanRun(t *testing.T) {
	deps := defaultDeps()
	addr, cancel, errs := startServe(t, NewRouter(deps), deps, noopHook, failingHook(errTearDown))
	waitForServer(t, addr)

	cancel()

	if err := waitResult(t, errs); !errors.Is(err, errTearDown) {
		t.Errorf("want errTearDown, got %v", err)
	}
}

func TestServeTearDownErrorKeepsTheFirstError(t *testing.T) {
	deps := withAddr(defaultDeps(), busyAddr(t))
	err := serve(context.Background(), deps, NewRouter(deps), noopHook, failingHook(errTearDown))

	if err == nil {
		t.Fatal("want the listen error, got nil")
	}
	if errors.Is(err, errTearDown) {
		t.Errorf("want the listen error, got the tear down error %v", err)
	}
}

func TestIgnoreServerClosed(t *testing.T) {
	other := errors.New("other")
	cases := map[string]struct {
		in   error
		want error
	}{
		"nil stays nil":             {nil, nil},
		"server closed becomes nil": {http.ErrServerClosed, nil},
		"wrapped server closed":     {errors.Join(http.ErrServerClosed, nil), nil},
		"other error stays":         {other, other},
	}
	for name, key := range cases {
		t.Run(name, func(t *testing.T) {
			if got := ignoreServerClosed(key.in); !errors.Is(got, key.want) {
				t.Errorf("want=%v, got=%v", key.want, got)
			}
		})
	}
}

func TestNewRouterRunsTheChain(t *testing.T) {
	repo := models.FakeDBRepo{}
	deps := defaultDeps()
	deps.dbRepo = &repo
	body := `{"update_id":1,"message":{"message_id":10,"message_thread_id":20,
	  "from":{"id":30},"chat":{"id":40},"text":"hello","date":1700000000}}`

	req := httptest.NewRequest(http.MethodPost, callbacks.TgWebhookPath, strings.NewReader(body))
	rec := httptest.NewRecorder()

	NewRouter(deps).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("want=%d, got=%d", http.StatusOK, rec.Code)
	}
	if len(repo.Messages) != 1 {
		t.Fatalf("want 1 stored message, got %d", len(repo.Messages))
	}
	if got := repo.Messages[0]; got.ID != 10 || got.ChatID != 40 || got.Text != "hello" {
		t.Errorf("want the message from the update, got %+v", got)
	}
}

func TestNewRouterRoutes(t *testing.T) {
	cases := map[string]struct {
		method string
		path   string
		want   int
	}{
		"telegram callback":     {http.MethodPost, "/v1/callbacks/telegram", http.StatusOK},
		"telegram wrong method": {http.MethodGet, "/v1/callbacks/telegram", http.StatusMethodNotAllowed},
		"unknown callback":      {http.MethodPost, "/v1/callbacks/unknown", http.StatusNotFound},
		"root path":             {http.MethodGet, "/", http.StatusNotFound},
	}
	for name, key := range cases {
		t.Run(name, func(t *testing.T) {
			req := httptest.NewRequest(key.method, key.path, strings.NewReader("{}"))
			rec := httptest.NewRecorder()

			NewRouter(defaultDeps()).ServeHTTP(rec, req)

			if rec.Code != key.want {
				t.Errorf("want=%d, got=%d", key.want, rec.Code)
			}
		})
	}
}
