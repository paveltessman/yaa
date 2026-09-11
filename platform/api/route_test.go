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
	"uuid"

	"github.com/paveltessman/yaa/pipelines/shared/ports/history"
	"github.com/paveltessman/yaa/pipelines/telegram/ports"
	"github.com/paveltessman/yaa/platform/api/callbacks"
	"github.com/paveltessman/yaa/platform/background"
	"github.com/paveltessman/yaa/platform/settings"
	historytestkit "github.com/paveltessman/yaa/platform/testkit/history"
	llmtestkit "github.com/paveltessman/yaa/platform/testkit/llm"
	"github.com/paveltessman/yaa/platform/testkit/telegram"
)

var errTearUp = errors.New("tear up failed")
var errTearDown = errors.New("tear down failed")

func noopHook(ctx context.Context, deps Deps) error { return nil }

func defaultDeps() Deps {
	s := settings.Settings{
		TgToken:    "12345:abcdefg",
		PublicHost: "http://example.com",
		ApiAddr:    "127.0.0.1:8080",
	}
	deps := Deps{
		settings:       &s,
		tgClient:       &telegram.FakeClient{},
		dbRepo:         &telegram.FakeDBRepo{},
		llmService:     &llmtestkit.FakeLLMService{Response: map[string]string{"text": "hi yourself"}},
		historyService: &historytestkit.FakeHistoryService{},
	}
	return deps

}

func withAddr(deps Deps, addr string) Deps {
	deps.settings.ApiAddr = addr
	return deps
}

func failingHook(err error) lifespan {
	return func(ctx context.Context, deps Deps) error { return err }
}

func countingHook(calls *int, err error) lifespan {
	return func(ctx context.Context, deps Deps) error {
		*calls++
		return err
	}
}

func liveCheckHook(ran *bool, ctxErr *error) lifespan {
	return func(ctx context.Context, deps Deps) error {
		*ran = true
		*ctxErr = ctx.Err()
		return nil
	}
}

func testRunner(t *testing.T) *background.Runner {
	t.Helper()
	runner := background.NewRunner(4, 5*time.Second)
	t.Cleanup(func() { waitRunner(t, runner) })
	return runner
}

func waitRunner(t *testing.T, runner *background.Runner) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := runner.Wait(ctx); err != nil {
		t.Errorf("want the background tasks to finish, got %v", err)
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

func startServe(
	t *testing.T,
	handler http.Handler,
	deps Deps,
	runner *background.Runner,
	tearUp, tearDown lifespan,
) (string, context.CancelFunc, <-chan error) {
	t.Helper()
	addr := freeAddr(t)
	deps = withAddr(deps, addr)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	errs := make(chan error, 1)
	go func() {
		errs <- serve(ctx, deps, handler, runner, tearUp, tearDown)
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
	runner := testRunner(t)
	addr, cancel, errs := startServe(t, NewRouter(deps, runner), deps, runner, noopHook, countingHook(&tearDownCalls, nil))
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
	runner := testRunner(t)
	addr, cancel, errs := startServe(t, NewRouter(deps, runner), deps, runner, noopHook, noopHook)
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

	addr, cancel, errs := startServe(t, handler, defaultDeps(), testRunner(t), noopHook, noopHook)
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
	runner := testRunner(t)
	err := serve(context.Background(), deps, NewRouter(deps, runner), runner, noopHook, countingHook(&tearDownCalls, nil))

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
	runner := testRunner(t)
	err := serve(context.Background(), deps, NewRouter(deps, runner), runner, failingHook(errTearUp), countingHook(&tearDownCalls, nil))

	if !errors.Is(err, errTearUp) {
		t.Errorf("want errTearUp, got %v", err)
	}
	if tearDownCalls != 0 {
		t.Errorf("want no tear down after a tear up failure, got %d calls", tearDownCalls)
	}
}

func TestServeTearDownErrorSurfacesAfterACleanRun(t *testing.T) {
	deps := defaultDeps()
	runner := testRunner(t)
	addr, cancel, errs := startServe(t, NewRouter(deps, runner), deps, runner, noopHook, failingHook(errTearDown))
	waitForServer(t, addr)

	cancel()

	if err := waitResult(t, errs); !errors.Is(err, errTearDown) {
		t.Errorf("want errTearDown, got %v", err)
	}
}

func TestServeTearDownErrorKeepsTheFirstError(t *testing.T) {
	deps := withAddr(defaultDeps(), busyAddr(t))
	runner := testRunner(t)
	err := serve(context.Background(), deps, NewRouter(deps, runner), runner, noopHook, failingHook(errTearDown))

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
	fromUser := ports.Message{
		ID:       10,
		ChatID:   40,
		ThreadID: 20,
		UserID:   30,
		Type:     ports.FromUser,
		Date:     time.Unix(1700000000, 0),
		Text:     "hello",
	}
	toUser := ports.Message{
		ID:       11,
		ChatID:   40,
		ThreadID: 20,
		UserID:   50,
		Type:     ports.ToUser,
		Date:     time.Unix(1700000001, 0),
		Text:     "hi yourself",
	}
	repo := telegram.FakeDBRepo{Thread: []*ports.Message{&fromUser}}
	client := telegram.FakeClient{SentMessage: &toUser}
	deps := defaultDeps()
	deps.dbRepo = &repo
	deps.tgClient = &client
	body := `{"update_id":1,"message":{"message_id":10,"message_thread_id":20,
	  "from":{"id":30},"chat":{"id":40},"text":"hello","date":1700000000}}`

	req := httptest.NewRequest(http.MethodPost, callbacks.TgWebhookPath, strings.NewReader(body))
	rec := httptest.NewRecorder()

	runner := background.NewRunner(1, 5*time.Second)
	NewRouter(deps, runner).ServeHTTP(rec, req)
	waitRunner(t, runner)

	if rec.Code != http.StatusOK {
		t.Errorf("want=%d, got=%d", http.StatusOK, rec.Code)
	}
	if len(repo.Messages) != 2 {
		t.Fatalf("want 2 stored messages, got %d", len(repo.Messages))
	}
	if got := repo.Messages[0]; got.ID != 10 || got.ChatID != 40 || got.Text != "hello" {
		t.Errorf("want the message from the update, got %+v", got)
	}
	if len(client.SendMessageCalls) != 1 {
		t.Fatalf("want 1 sent message, got %d", len(client.SendMessageCalls))
	}
	if got := client.SendMessageCalls[0]; got.ChatID != 40 || got.Text != "hi yourself" {
		t.Errorf("want the reply of the agent, got %+v", got)
	}
}

func recordOf(t *testing.T, saver *historytestkit.FakeHistoryService, pipeline string) *history.Record {
	t.Helper()

	for _, record := range saver.Records {
		if record.Pipeline == pipeline {
			return record
		}
	}
	t.Fatalf("want a record of the %s pipeline, got none", pipeline)
	return nil
}

func TestNewRouterStoresTheHistoryOfEveryPass(t *testing.T) {
	saver := historytestkit.FakeHistoryService{}
	deps := defaultDeps()
	deps.dbRepo = &telegram.FakeDBRepo{Thread: []*ports.Message{{Type: ports.FromUser, Text: "hello"}}}
	deps.tgClient = &telegram.FakeClient{SentMessage: &ports.Message{ID: 11, ChatID: 40}}
	deps.historyService = &saver
	body := `{"update_id":1,"message":{"message_id":10,"message_thread_id":20,
	  "from":{"id":30},"chat":{"id":40},"text":"hello","date":1700000000}}`

	req := httptest.NewRequest(http.MethodPost, callbacks.TgWebhookPath, strings.NewReader(body))
	runner := background.NewRunner(1, 5*time.Second)
	NewRouter(deps, runner).ServeHTTP(httptest.NewRecorder(), req)
	waitRunner(t, runner)

	if saver.Calls != 2 {
		t.Fatalf("want 2 saves, got %d", saver.Calls)
	}
	update := recordOf(t, &saver, "telegram.updates")
	agent := recordOf(t, &saver, "agent")
	if update.ParentID != uuid.Nil() {
		t.Errorf("the update pass: want no parent, got %s", update.ParentID)
	}
	if agent.ParentID != update.SessionID {
		t.Errorf("the agent pass: parent id: want=%s, got=%s", update.SessionID, agent.ParentID)
	}
	if agent.SessionID == update.SessionID {
		t.Error("want a session of its own for the agent pass, got the session of the update pass")
	}
	if update.Date.IsZero() || agent.Date.IsZero() {
		t.Error("want a date on both passes, got the zero time")
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
		"history list":          {http.MethodGet, "/history/", http.StatusOK},
		"history wrong method":  {http.MethodPost, "/history/", http.StatusMethodNotAllowed},
		"history bad session":   {http.MethodGet, "/history/not-a-uuid", http.StatusBadRequest},
		"root path":             {http.MethodGet, "/", http.StatusNotFound},
	}
	for name, key := range cases {
		t.Run(name, func(t *testing.T) {
			req := httptest.NewRequest(key.method, key.path, strings.NewReader("{}"))
			rec := httptest.NewRecorder()

			NewRouter(defaultDeps(), testRunner(t)).ServeHTTP(rec, req)

			if rec.Code != key.want {
				t.Errorf("want=%d, got=%d", key.want, rec.Code)
			}
		})
	}
}

func TestServeTearsDownWithALiveContext(t *testing.T) {
	var ran bool
	var ctxErr error
	deps := defaultDeps()
	runner := testRunner(t)
	addr, cancel, errs := startServe(t, NewRouter(deps, runner), deps, runner, noopHook, liveCheckHook(&ran, &ctxErr))
	waitForServer(t, addr)

	cancel()

	if err := waitResult(t, errs); err != nil {
		t.Errorf("want no error on a clean shutdown, got %v", err)
	}
	if !ran {
		t.Fatal("want a tear down call, got none")
	}
	if ctxErr != nil {
		t.Errorf("want a live context for tear down, got %v", ctxErr)
	}
}

func TestTearUpAndTearDownPassTheContext(t *testing.T) {
	type key struct{}
	client := telegram.FakeClient{}
	deps := defaultDeps()
	deps.tgClient = &client
	ctx := context.WithValue(t.Context(), key{}, "marker")

	if err := tearUp(ctx, deps); err != nil {
		t.Fatalf("want no error, got %v", err)
	}
	if err := tearDown(ctx, deps); err != nil {
		t.Fatalf("want no error, got %v", err)
	}

	if len(client.Contexts) != 2 {
		t.Fatalf("want 2 contexts, got %d", len(client.Contexts))
	}
	for i, got := range client.Contexts {
		if value := got.Value(key{}); value != "marker" {
			t.Errorf("context %d: want=%q, got=%v", i, "marker", value)
		}
	}
}
