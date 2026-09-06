package telegram

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/paveltessman/yaa/pipelines/telegram/ports"
	"github.com/paveltessman/yaa/platform/network"
)

var errTransport = errors.New("transport is down")

type fakeRequester struct {
	resp []byte
	err  error

	calls   int
	gotCtx  context.Context
	gotPath string
	gotType network.ContentType
	gotBody []byte
}

var _ network.HTTPRequester = (*fakeRequester)(nil)

func (f *fakeRequester) GetBytes(ctx context.Context, path string) ([]byte, error) {
	f.calls++
	f.gotCtx = ctx
	f.gotPath = path
	return f.resp, f.err
}

func (f *fakeRequester) PostBytes(ctx context.Context, path string, contentType network.ContentType, body []byte) ([]byte, error) {
	f.calls++
	f.gotCtx = ctx
	f.gotPath = path
	f.gotType = contentType
	f.gotBody = body
	return f.resp, f.err
}

func newTestClient(t *testing.T, resp string, err error) (*fakeRequester, *Client) {
	t.Helper()
	f := &fakeRequester{resp: []byte(resp), err: err}
	return f, NewClient(f)
}

func TestNewSession(t *testing.T) {
	cases := map[string]struct {
		token       string
		shouldPanic bool
	}{
		"plain token": {"123456:abc-def", false},
		"empty token": {"", true},
	}
	for name, key := range cases {
		t.Run(name, func(t *testing.T) {
			defer func() {
				err := recover()
				if key.shouldPanic && err == nil {
					t.Errorf("NewSession accepted %s", name)
				}
				if !key.shouldPanic && err != nil {
					t.Errorf("want no panic, got %v", err)
				}
			}()

			if got := NewSession(key.token); got == nil && !key.shouldPanic {
				t.Errorf("want a session, got nil")
			}
		})
	}
}

func TestNewClientKeepsSession(t *testing.T) {
	f := &fakeRequester{resp: []byte(`{"ok":true}`)}

	c := NewClient(f)

	if c == nil {
		t.Fatalf("want a client, got nil")
	}
	if c.http != network.HTTPRequester(f) {
		t.Errorf("want the given session, got %v", c.http)
	}
}

func TestGetMeRequest(t *testing.T) {
	f, c := newTestClient(t, `{"ok":true,"result":{"id":1,"is_bot":true}}`, nil)

	if _, err := c.GetMe(t.Context()); err != nil {
		t.Fatalf("want no error, got %v", err)
	}

	if f.calls != 1 {
		t.Errorf("want 1 call, got %d", f.calls)
	}
	if want := "/getMe"; f.gotPath != want {
		t.Errorf("want=%q, got=%q", want, f.gotPath)
	}
	if f.gotType != network.ApplicationJson {
		t.Errorf("want content type %q, got %q", network.ApplicationJson, f.gotType)
	}
	if want := "null"; string(f.gotBody) != want {
		t.Errorf("want body %q, got %q", want, f.gotBody)
	}
}

func TestSetWebhookRequest(t *testing.T) {
	cases := map[string]struct{ url, wantBody string }{
		"plain url": {"https://example.com/hook", `{"url":"https://example.com/hook","allowed_updates":["message"]}`},
	}
	for name, key := range cases {
		t.Run(name, func(t *testing.T) {
			f, c := newTestClient(t, `{"ok":true}`, nil)

			params := ports.SetWebhookParams{
				URL:            key.url,
				AllowedUpdates: []string{"message"},
			}

			if err := c.SetWebhook(t.Context(), params); err != nil {
				t.Fatalf("want no error, got %v", err)
			}

			if f.calls != 1 {
				t.Errorf("want 1 call, got %d", f.calls)
			}
			if want := "/setWebhook"; f.gotPath != want {
				t.Errorf("want=%q, got=%q", want, f.gotPath)
			}
			if f.gotType != network.ApplicationJson {
				t.Errorf("want content type %q, got %q", network.ApplicationJson, f.gotType)
			}
			if string(f.gotBody) != key.wantBody {
				t.Errorf("want body %q, got %q", key.wantBody, f.gotBody)
			}
		})
	}
}

func TestDeleteWebhookRequest(t *testing.T) {
	f, c := newTestClient(t, `{"ok":true}`, nil)

	if err := c.DeleteWebhook(t.Context()); err != nil {
		t.Fatalf("want no error, got %v", err)
	}

	if f.calls != 1 {
		t.Errorf("want 1 call, got %d", f.calls)
	}
	if want := "/deleteWebhook"; f.gotPath != want {
		t.Errorf("want=%q, got=%q", want, f.gotPath)
	}
	if f.gotType != network.ApplicationJson {
		t.Errorf("want content type %q, got %q", network.ApplicationJson, f.gotType)
	}
	if want := "null"; string(f.gotBody) != want {
		t.Errorf("want body %q, got %q", want, f.gotBody)
	}
}

var callers = map[string]func(context.Context, *Client) error{
	"GetMe": func(ctx context.Context, c *Client) error { _, err := c.GetMe(ctx); return err },
	"SetWebhook": func(ctx context.Context, c *Client) error {
		return c.SetWebhook(ctx, ports.SetWebhookParams{URL: "https://example.com/hook"})
	},
	"DeleteWebhook": func(ctx context.Context, c *Client) error { return c.DeleteWebhook(ctx) },
}

func TestRequestTransportError(t *testing.T) {
	for name, call := range callers {
		t.Run(name, func(t *testing.T) {
			_, c := newTestClient(t, "", errTransport)

			err := call(t.Context(), c)

			if !errors.Is(err, errTransport) {
				t.Errorf("want errTransport, got %v", err)
			}
			if errors.Is(err, ErrTelegramAPIFailed) {
				t.Errorf("want no ErrTelegramAPIFailed on a transport error, got %v", err)
			}
		})
	}
}

func TestRequestBadJSON(t *testing.T) {
	cases := map[string]string{
		"not json":       "not json at all",
		"empty body":     "",
		"truncated json": `{"ok":true`,
		"wrong type":     `{"ok":"yes"}`,
		"array body":     `[1,2,3]`,
	}
	for name, resp := range cases {
		for caller, call := range callers {
			t.Run(name+"/"+caller, func(t *testing.T) {
				_, c := newTestClient(t, resp, nil)

				err := call(t.Context(), c)

				if err == nil {
					t.Fatalf("want an error, got nil")
				}
				if errors.Is(err, ErrTelegramAPIFailed) {
					t.Errorf("want a json error, got %v", err)
				}
			})
		}
	}
}

func TestRequestAPIFailure(t *testing.T) {
	cases := map[string]string{
		"ok is false":  `{"ok":false,"description":"unauthorized"}`,
		"ok is absent": `{"description":"bad request"}`,
		"empty object": `{}`,
	}
	for name, resp := range cases {
		for caller, call := range callers {
			t.Run(name+"/"+caller, func(t *testing.T) {
				_, c := newTestClient(t, resp, nil)

				err := call(t.Context(), c)

				if !errors.Is(err, ErrTelegramAPIFailed) {
					t.Fatalf("want ErrTelegramAPIFailed, got %v", err)
				}
				if !strings.Contains(err.Error(), resp) {
					t.Errorf("want the raw body %q in %q", resp, err)
				}
			})
		}
	}
}

func TestRequestPassesContext(t *testing.T) {
	type key struct{}
	for name, call := range callers {
		t.Run(name, func(t *testing.T) {
			f, c := newTestClient(t, `{"ok":true}`, nil)
			ctx := context.WithValue(t.Context(), key{}, "marker")

			if err := call(ctx, c); err != nil {
				t.Fatalf("want no error, got %v", err)
			}
			if f.gotCtx == nil {
				t.Fatal("want a context, got nil")
			}
			if got := f.gotCtx.Value(key{}); got != "marker" {
				t.Errorf("want=%q, got=%v", "marker", got)
			}
		})
	}
}
