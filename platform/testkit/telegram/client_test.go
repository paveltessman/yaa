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
	if _, err := c.SendMessage(ctx, ports.SendMessageParams{ChatID: 40, Text: "hello"}); err != nil {
		t.Fatalf("want no error, got %v", err)
	}

	if got := c.CallCounts["SetWebhook"]; got != 1 {
		t.Errorf("want 1 SetWebhook call, got %d", got)
	}
	if got := c.CallCounts["DeleteWebhook"]; got != 1 {
		t.Errorf("want 1 DeleteWebhook call, got %d", got)
	}
	if got := c.CallCounts["SendMessage"]; got != 1 {
		t.Errorf("want 1 SendMessage call, got %d", got)
	}
	if len(c.Contexts) != 3 {
		t.Fatalf("want 3 contexts, got %d", len(c.Contexts))
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
	got, err := c.SendMessage(t.Context(), ports.SendMessageParams{})
	if !errors.Is(err, wantErr) {
		t.Errorf("want the fake error, got %v", err)
	}
	if got != nil {
		t.Errorf("want no message on error, got %+v", got)
	}
}

func TestFakeClientKeepsSendMessageParams(t *testing.T) {
	sent := ports.Message{ID: 10, ChatID: 40, Type: ports.ToUser, Text: "hello"}
	c := FakeClient{SentMessage: &sent}
	params := ports.SendMessageParams{ChatID: 40, ThreadID: 20, Text: "hello"}

	got, err := c.SendMessage(t.Context(), params)

	if err != nil {
		t.Fatalf("want no error, got %v", err)
	}
	if got != &sent {
		t.Errorf("want the given message, got %+v", got)
	}
	if len(c.SendMessageCalls) != 1 {
		t.Fatalf("want 1 recorded call, got %d", len(c.SendMessageCalls))
	}
	if c.SendMessageCalls[0] != params {
		t.Errorf("want=%+v, got=%+v", params, c.SendMessageCalls[0])
	}
}
