package callbacks

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/paveltessman/yaa/pipelines/shared"
	"github.com/paveltessman/yaa/pipelines/telegram/updates/session"
)

func quietLog(t *testing.T) {
	t.Helper()
	log.SetOutput(io.Discard)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })
}

func handler() http.Handler {
	h := Telegram(shared.Run, shared.Chain[*session.Session]{})
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

			handler().ServeHTTP(rec, req)

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

			handler().ServeHTTP(rec, req)

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

	handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("want=%d, got=%d", http.StatusBadRequest, rec.Code)
	}
}
