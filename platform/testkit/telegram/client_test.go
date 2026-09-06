package telegram

import (
	"context"
	"errors"
	"testing"

	"github.com/paveltessman/yaa/pipelines/telegram/ports"
)

func TestFakeClientRecordsCalls(t *testing.T) {
	type key struct{}
	c := FakeClient{}
	ctx := context.WithValue(t.Context(), key{}, "marker")

	if err := c.SetWebhook(ctx, ports.SetWebhookParams{}); err != nil {
		t.Fatalf("want no error, got %v", err)
	}
	if err := c.DeleteWebhook(ctx); err != nil {
		t.Fatalf("want no error, got %v", err)
	}

	if got := c.CallCounts["SetWebhook"]; got != 1 {
		t.Errorf("want 1 SetWebhook call, got %d", got)
	}
	if got := c.CallCounts["DeleteWebhook"]; got != 1 {
		t.Errorf("want 1 DeleteWebhook call, got %d", got)
	}
	if len(c.Contexts) != 2 {
		t.Fatalf("want 2 contexts, got %d", len(c.Contexts))
	}
	for i, ctx := range c.Contexts {
		if got := ctx.Value(key{}); got != "marker" {
			t.Errorf("context %d: want=%q, got=%v", i, "marker", got)
		}
	}
}

func TestFakeClientReturnsItsError(t *testing.T) {
	wantErr := errors.New("telegram is down")
	c := FakeClient{Error: wantErr}

	if err := c.SetWebhook(t.Context(), ports.SetWebhookParams{}); !errors.Is(err, wantErr) {
		t.Errorf("want the fake error, got %v", err)
	}
	if err := c.DeleteWebhook(t.Context()); !errors.Is(err, wantErr) {
		t.Errorf("want the fake error, got %v", err)
	}
}
