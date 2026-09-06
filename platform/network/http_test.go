package network

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

const testTimeout = 5 * time.Second

func newTestSession(t *testing.T, prefix string, h http.HandlerFunc) (*httptest.Server, *HTTPSession) {
	t.Helper()
	return newTestSessionTimeout(t, prefix, testTimeout, h)
}

func newTestSessionTimeout(t *testing.T, prefix string, timeout time.Duration, h http.HandlerFunc) (*httptest.Server, *HTTPSession) {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return srv, NewHTTPSession(srv.URL+prefix, timeout)
}

func TestGetBytesStatus(t *testing.T) {
	cases := map[string]struct {
		status  int
		wantErr bool
	}{
		"200 lowest success":  {http.StatusOK, false},
		"201 not just 200":    {http.StatusCreated, false},
		"299 highest success": {299, false},
		"300 first failure":   {http.StatusMultipleChoices, true},
		"404 not found":       {http.StatusNotFound, true},
		"500 server error":    {http.StatusInternalServerError, true},
	}
	for name, key := range cases {
		t.Run(name, func(t *testing.T) {
			_, s := newTestSession(t, "", func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(key.status)
				_, _ = w.Write([]byte("payload"))
			})

			got, err := s.GetBytes(t.Context(), "/path")

			if key.wantErr {
				if !errors.Is(err, ErrHTTPFailed) {
					t.Errorf("want ErrHTTPFailed, got %v", err)
				}
				if got != nil {
					t.Errorf("want no body on error, got %q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("want no error, got %v", err)
			}
			if want := "payload"; string(got) != want {
				t.Errorf("want=%q, got=%q", want, got)
			}
		})
	}
}

func TestGetBytesBodySize(t *testing.T) {
	cases := map[string]struct {
		size    int
		wantErr bool
	}{
		"empty body":          {0, false},
		"small body":          {10, false},
		"exactly at limit":    {maxBodyBytes, false},
		"one byte over limit": {maxBodyBytes + 1, true},
	}
	for name, key := range cases {
		t.Run(name, func(t *testing.T) {
			_, s := newTestSession(t, "", func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write(bytes.Repeat([]byte("a"), key.size))
			})

			got, err := s.GetBytes(t.Context(), "data")

			if key.wantErr {
				if !errors.Is(err, ErrHTTPFailed) {
					t.Errorf("want ErrHTTPFailed, got %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("want no error, got %v", err)
			}
			if len(got) != key.size {
				t.Errorf("want %d bytes, got %d", key.size, len(got))
			}
		})
	}
}

func TestGetBytesRequestURI(t *testing.T) {
	cases := map[string]struct{ prefix, path, want string }{
		"plain path":          {"", "data", "/data"},
		"leading slash":       {"", "/data", "/data"},
		"nested path":         {"", "some/data", "/some/data"},
		"space is escaped":    {"", "some file", "/some%20file"},
		"parent dir dropped":  {"", "../data", "/data"},
		"base keeps its path": {"/api", "data", "/api/data"},
	}
	for name, key := range cases {
		t.Run(name, func(t *testing.T) {
			var gotURI string
			_, s := newTestSession(t, key.prefix, func(w http.ResponseWriter, r *http.Request) {
				gotURI = r.RequestURI
			})

			if _, err := s.GetBytes(t.Context(), key.path); err != nil {
				t.Fatalf("want no error, got %v", err)
			}
			if gotURI != key.want {
				t.Errorf("want=%q, got=%q", key.want, gotURI)
			}
		})
	}
}

func TestGetBytesTransportFailure(t *testing.T) {
	t.Run("no server listening", func(t *testing.T) {
		srv, s := newTestSession(t, "", http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
		srv.Close()

		if _, err := s.GetBytes(t.Context(), "data"); !errors.Is(err, ErrHTTPFailed) {
			t.Errorf("want ErrHTTPFailed, got %v", err)
		}
	})

	t.Run("body ends early", func(t *testing.T) {
		_, s := newTestSession(t, "", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Length", "5000")
			_, _ = w.Write([]byte("short"))
			w.(http.Flusher).Flush()
			panic(http.ErrAbortHandler)
		})

		_, err := s.GetBytes(t.Context(), "data")
		if !errors.Is(err, ErrHTTPFailed) {
			t.Fatalf("want ErrHTTPFailed, got %v", err)
		}
		if strings.Contains(err.Error(), "\x00") {
			t.Errorf("error message carries unread buffer bytes: %q", err)
		}
	})
}

func TestPrepareBaseURL(t *testing.T) {
	cases := map[string]struct {
		url, want   string
		shouldPanic bool
	}{
		"fine url":                       {"https://google.com", "https://google.com", false},
		"fine url with path":             {"https://google.com/some", "https://google.com/some", false},
		"fine url with unescaped path":   {"https://google.com/some other", "https://google.com/some%20other", false},
		"fine url with first-level host": {"http://postgres", "http://postgres", false},
		"fine url with trailing slash":   {"https://google.com/", "https://google.com", false},
		"empty url":                      {"", "", true},
		"mailformed url":                 {"https:/google.com/", "", true},
		"url without scheme":             {"google.com", "", true},
		"url without host":               {"https://", "", true},
		"unsupported scheme":             {"wss://example.com", "", true},
	}
	for name, key := range cases {
		t.Run(name, func(t *testing.T) {
			defer func() {
				if !key.shouldPanic {
					return
				}
				err := recover()
				if err == nil {
					t.Errorf("prepareBaseURL accepted %s", name)
				}
			}()

			if got, want := prepareBaseURL(key.url), key.want; got != want {
				t.Errorf("%s: want=%q, got=%q", name, want, got)
			}
		})
	}
}

func TestPreparePath(t *testing.T) {
	cases := map[string]struct{ path, want string }{
		"single-segment, no slash at start, no slash at end": {"some-path", "some-path"},
		"single-segment, slash at start, no slash at end":    {"/some-path", "some-path"},
		"single-segment, no slash at start, slash at end":    {"some-path/", "some-path/"},
		"single-segment, slash at start, slash at end":       {"/some-path/", "some-path/"},
		"multi-segment, no slash at start, no slash at end":  {"some-other/path", "some-other/path"},
		"multi-segment, slash at start, no slash at end":     {"/some-other/path", "some-other/path"},
		"multi-segment, no slash at start, slash at end":     {"some-other/path/", "some-other/path/"},
		"multi-segment, slash at start, slash at end":        {"/some-other/path/", "some-other/path/"},
		"multi-segment, not escaped":                         {"/some other/path/", "some%20other/path/"},
		"multi-segment with with double dots":                {"../some-other/path/", "some-other/path/"},
	}
	for name, key := range cases {
		t.Run(name, func(*testing.T) {
			if got, want := preparePath(key.path), key.want; got != want {
				t.Errorf("want=%q, got=%q", want, got)
			}
		})
	}
}

func TestPostBytesRequest(t *testing.T) {
	cases := map[string]struct {
		contentType ContentType
		body        string
	}{
		"json body":     {"application/json", `{"url":"https://example.com"}`},
		"text body":     {"text/plain", "hello"},
		"empty body":    {"application/json", ""},
		"no media type": {"", "raw"},
	}
	for name, key := range cases {
		t.Run(name, func(t *testing.T) {
			var gotMethod, gotType string
			var gotBody []byte
			_, s := newTestSession(t, "", func(w http.ResponseWriter, r *http.Request) {
				gotMethod = r.Method
				gotType = r.Header.Get("Content-Type")
				gotBody, _ = io.ReadAll(r.Body)
				_, _ = w.Write([]byte("answer"))
			})

			got, err := s.PostBytes(t.Context(), "data", key.contentType, []byte(key.body))
			if err != nil {
				t.Fatalf("want no error, got %v", err)
			}

			if gotMethod != http.MethodPost {
				t.Errorf("want method %q, got %q", http.MethodPost, gotMethod)
			}
			if gotType != string(key.contentType) {
				t.Errorf("want content type %q, got %q", key.contentType, gotType)
			}
			if string(gotBody) != key.body {
				t.Errorf("want body %q, got %q", key.body, gotBody)
			}
			if want := "answer"; string(got) != want {
				t.Errorf("want=%q, got=%q", want, got)
			}
		})
	}
}

func TestPostBytesNilBody(t *testing.T) {
	var gotLength int64
	_, s := newTestSession(t, "", func(w http.ResponseWriter, r *http.Request) {
		gotLength = r.ContentLength
	})

	if _, err := s.PostBytes(t.Context(), "data", "application/json", nil); err != nil {
		t.Fatalf("want no error, got %v", err)
	}
	if gotLength != 0 {
		t.Errorf("want no body, got %d bytes", gotLength)
	}
}

func TestPostBytesStatus(t *testing.T) {
	cases := map[string]struct {
		status  int
		wantErr bool
	}{
		"200 success":      {http.StatusOK, false},
		"400 bad request":  {http.StatusBadRequest, true},
		"500 server error": {http.StatusInternalServerError, true},
	}
	for name, key := range cases {
		t.Run(name, func(t *testing.T) {
			_, s := newTestSession(t, "", func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(key.status)
				_, _ = w.Write([]byte("payload"))
			})

			got, err := s.PostBytes(t.Context(), "data", "application/json", []byte("{}"))

			if key.wantErr {
				if !errors.Is(err, ErrHTTPFailed) {
					t.Errorf("want ErrHTTPFailed, got %v", err)
				}
				if got != nil {
					t.Errorf("want no body on error, got %q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("want no error, got %v", err)
			}
		})
	}
}

func TestPostBytesRequestURI(t *testing.T) {
	cases := map[string]struct{ prefix, path, want string }{
		"plain path":          {"", "data", "/data"},
		"leading slash":       {"", "/data", "/data"},
		"space is escaped":    {"", "some file", "/some%20file"},
		"parent dir dropped":  {"", "../data", "/data"},
		"base keeps its path": {"/api", "data", "/api/data"},
	}
	for name, key := range cases {
		t.Run(name, func(t *testing.T) {
			var gotURI string
			_, s := newTestSession(t, key.prefix, func(w http.ResponseWriter, r *http.Request) {
				gotURI = r.RequestURI
			})

			if _, err := s.PostBytes(t.Context(), key.path, "application/json", []byte("{}")); err != nil {
				t.Fatalf("want no error, got %v", err)
			}
			if gotURI != key.want {
				t.Errorf("want=%q, got=%q", key.want, gotURI)
			}
		})
	}
}

func TestPostBytesTransportFailure(t *testing.T) {
	srv, s := newTestSession(t, "", http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	srv.Close()

	if _, err := s.PostBytes(t.Context(), "data", "application/json", []byte("{}")); !errors.Is(err, ErrHTTPFailed) {
		t.Errorf("want ErrHTTPFailed, got %v", err)
	}
}

func TestNewHTTPSessionTimeout(t *testing.T) {
	cases := map[string]struct {
		timeout     time.Duration
		shouldPanic bool
	}{
		"a second":      {time.Second, false},
		"a millisecond": {time.Millisecond, false},
		"zero":          {0, true},
		"negative":      {-time.Second, true},
	}
	for name, key := range cases {
		t.Run(name, func(t *testing.T) {
			defer func() {
				err := recover()
				if key.shouldPanic && err == nil {
					t.Errorf("NewHTTPSession accepted %s", name)
				}
				if !key.shouldPanic && err != nil {
					t.Errorf("want no panic, got %v", err)
				}
			}()

			s := NewHTTPSession("https://example.com", key.timeout)
			if s.client.Timeout != key.timeout {
				t.Errorf("want timeout %v, got %v", key.timeout, s.client.Timeout)
			}
		})
	}
}

func TestRequestsHonourACancelledContext(t *testing.T) {
	var calls atomic.Int64
	_, s := newTestSession(t, "", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
	})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	cases := map[string]func() ([]byte, error){
		"GetBytes":  func() ([]byte, error) { return s.GetBytes(ctx, "data") },
		"PostBytes": func() ([]byte, error) { return s.PostBytes(ctx, "data", ApplicationJson, []byte("{}")) },
	}
	for name, call := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := call()

			if !errors.Is(err, ErrHTTPFailed) {
				t.Errorf("want ErrHTTPFailed, got %v", err)
			}
			if !errors.Is(err, context.Canceled) {
				t.Errorf("want context.Canceled, got %v", err)
			}
		})
	}
	if got := calls.Load(); got != 0 {
		t.Errorf("want no call to reach the server, got %d", got)
	}
}

func TestRequestsStopOnTheTimeout(t *testing.T) {
	release := make(chan struct{})
	_, s := newTestSessionTimeout(t, "", 50*time.Millisecond, func(w http.ResponseWriter, r *http.Request) {
		<-release
	})
	t.Cleanup(func() { close(release) })

	cases := map[string]func() ([]byte, error){
		"GetBytes":  func() ([]byte, error) { return s.GetBytes(t.Context(), "data") },
		"PostBytes": func() ([]byte, error) { return s.PostBytes(t.Context(), "data", ApplicationJson, []byte("{}")) },
	}
	for name, call := range cases {
		t.Run(name, func(t *testing.T) {
			got, err := call()

			if !errors.Is(err, ErrHTTPFailed) {
				t.Errorf("want ErrHTTPFailed, got %v", err)
			}
			if got != nil {
				t.Errorf("want no body on a timeout, got %q", got)
			}
		})
	}
}

func TestNewRequestCarriesTheContext(t *testing.T) {
	type key struct{}
	s := NewHTTPSession("https://example.com", testTimeout)
	ctx := context.WithValue(t.Context(), key{}, "marker")

	req, err := s.newRequest(ctx, http.MethodGet, "data", "", nil)

	if err != nil {
		t.Fatalf("want no error, got %v", err)
	}
	if got := req.Context().Value(key{}); got != "marker" {
		t.Errorf("want=%q, got=%v", "marker", got)
	}
}
