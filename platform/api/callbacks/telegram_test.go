package callbacks

import (
	"bytes"
	"context"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/paveltessman/yaa/pipelines/shared"
	"github.com/paveltessman/yaa/pipelines/shared/ports/history"
	"github.com/paveltessman/yaa/pipelines/telegram/updates/session"
	"github.com/paveltessman/yaa/platform/background"
	testkit "github.com/paveltessman/yaa/platform/testkit/history"
)

func fakeHistoryService() history.HistoryService {
	return &testkit.FakeHistoryService{}
}

func quietLog(t *testing.T) {
	t.Helper()
	log.SetOutput(io.Discard)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })
}

func testRunner(t *testing.T) *background.Runner {
	t.Helper()
	runner := background.NewRunner(4, 5*time.Second)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := runner.Wait(ctx); err != nil {
			t.Errorf("want the background tasks to finish, got %v", err)
		}
	})
	return runner
}

func handler(t *testing.T) http.Handler {
	t.Helper()
	pipeline := shared.NewPipeline(fakeHistoryService(), shared.Chain[*session.Session]{})
	h := Telegram(pipeline, testRunner(t))
	return h
}

func TestTelegramMethod(t *testing.T) {
	cases := map[string]struct {
		method string
		want   int
	}{
		"post is allowed":    {http.MethodPost, http.StatusOK},
		"get is rejected":    {http.MethodGet, http.StatusMethodNotAllowed},
		"put is rejected":    {http.MethodPut, http.StatusMethodNotAllowed},
		"delete is rejected": {http.MethodDelete, http.StatusMethodNotAllowed},
	}
	for name, key := range cases {
		t.Run(name, func(t *testing.T) {
			quietLog(t)
			req := httptest.NewRequest(key.method, "/v1/callbacks/telegram", strings.NewReader(`{"update_id":1}`))
			rec := httptest.NewRecorder()

			handler(t).ServeHTTP(rec, req)

			if rec.Code != key.want {
				t.Errorf("want=%d, got=%d", key.want, rec.Code)
			}
			if key.want != http.StatusMethodNotAllowed {
				return
			}
			if got := rec.Header().Get("Allow"); got != http.MethodPost {
				t.Errorf("want Allow=%q, got %q", http.MethodPost, got)
			}
		})
	}
}

func TestTelegramBodySize(t *testing.T) {
	cases := map[string]struct {
		size int
		want int
	}{
		"empty body":          {0, http.StatusOK},
		"small body":          {10, http.StatusOK},
		"exactly at limit":    {maxBodyBytes, http.StatusOK},
		"one byte over limit": {maxBodyBytes + 1, http.StatusBadRequest},
	}
	for name, key := range cases {
		t.Run(name, func(t *testing.T) {
			quietLog(t)
			body := bytes.NewReader(bytes.Repeat([]byte("a"), key.size))
			req := httptest.NewRequest(http.MethodPost, "/v1/callbacks/telegram", body)
			rec := httptest.NewRecorder()

			handler(t).ServeHTTP(rec, req)

			if rec.Code != key.want {
				t.Errorf("want=%d, got=%d", key.want, rec.Code)
			}
		})
	}
}

type errReader struct{}

func (errReader) Read([]byte) (int, error) { return 0, io.ErrUnexpectedEOF }

func TestTelegramReadErrorGivesBadRequest(t *testing.T) {
	quietLog(t)
	req := httptest.NewRequest(http.MethodPost, "/v1/callbacks/telegram", errReader{})
	rec := httptest.NewRecorder()

	handler(t).ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("want=%d, got=%d", http.StatusBadRequest, rec.Code)
	}
}

func TestTelegramGivesTheUpdateToThePipeline(t *testing.T) {
	runner := testRunner(t)
	bodies := make(chan string, 1)
	errs := make(chan error, 1)

	pipeline := func(ctx context.Context, s *session.Session) error {
		bodies <- string(s.RawUpdate)
		errs <- ctx.Err()
		return nil
	}
	body := `{"update_id":7}`
	req := httptest.NewRequest(http.MethodPost, TgWebhookPath, strings.NewReader(body))
	rec := httptest.NewRecorder()

	Telegram(pipeline, runner).ServeHTTP(rec, req)

	select {
	case got := <-bodies:
		if got != body {
			t.Errorf("want=%q, got=%q", body, got)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("want the pipeline to run, got no call")
	}
	if err := <-errs; err != nil {
		t.Errorf("want a live context after the answer, got %v", err)
	}
}
