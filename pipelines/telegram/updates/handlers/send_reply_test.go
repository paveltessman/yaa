package handlers

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/paveltessman/yaa/pipelines/shared"
	. "github.com/paveltessman/yaa/pipelines/telegram/ports"
	"github.com/paveltessman/yaa/pipelines/telegram/updates/session"
	"github.com/paveltessman/yaa/platform/testkit/telegram"
)

func newSentMessage() *Message {
	message := Message{
		ID:       11,
		ChatID:   40,
		ThreadID: 20,
		UserID:   50,
		Type:     ToUser,
		Date:     time.Unix(1700000001, 0),
		Text:     "hi yourself",
	}
	return &message
}

func newSessionWithReply(message *Message, reply string) *session.Session {
	s := newSessionWithMessage(message)
	s.Reply = reply
	return s
}

func TestSendReplySendsTheReplyToTheChatOfTheMessage(t *testing.T) {
	sent := newSentMessage()
	client := telegram.FakeClient{SentMessage: sent}
	h := NewSendReply(&client)
	message := newMessage()
	s := newSessionWithReply(message, "hi yourself")

	if err := h.Handle(context.Background(), s); err != nil {
		t.Fatalf("want no error, got %v", err)
	}
	if got := client.CallCounts["SendMessage"]; got != 1 {
		t.Errorf("want 1 call, got %d", got)
	}
	if len(client.SendMessageCalls) != 1 {
		t.Fatalf("want 1 call, got %d", len(client.SendMessageCalls))
	}
	want := SendMessageParams{ChatID: message.ChatID, ThreadID: message.ThreadID, Text: "hi yourself"}
	if got := client.SendMessageCalls[0]; got != want {
		t.Errorf("want=%+v, got=%+v", want, got)
	}
	if s.SentMessage != sent {
		t.Errorf("want=%+v, got=%+v", sent, s.SentMessage)
	}
}

func TestSendReplySendsToAChatWithoutTopics(t *testing.T) {
	client := telegram.FakeClient{SentMessage: newSentMessage()}
	h := NewSendReply(&client)
	message := newMessage()
	message.ThreadID = 0
	s := newSessionWithReply(message, "hi yourself")

	if err := h.Handle(context.Background(), s); err != nil {
		t.Fatalf("want no error, got %v", err)
	}
	if len(client.SendMessageCalls) != 1 {
		t.Fatalf("want 1 call, got %d", len(client.SendMessageCalls))
	}
	want := SendMessageParams{ChatID: message.ChatID, ThreadID: 0, Text: "hi yourself"}
	if got := client.SendMessageCalls[0]; got != want {
		t.Errorf("want=%+v, got=%+v", want, got)
	}
}

func TestSendReplyKeepsMessageOnSession(t *testing.T) {
	client := telegram.FakeClient{SentMessage: newSentMessage()}
	h := NewSendReply(&client)
	message := newMessage()
	s := newSessionWithReply(message, "hi yourself")

	if err := h.Handle(context.Background(), s); err != nil {
		t.Fatalf("want no error, got %v", err)
	}
	if s.Message != message {
		t.Errorf("want=%+v, got=%+v", message, s.Message)
	}
}

func TestSendReplyPassesContext(t *testing.T) {
	type key struct{}
	client := telegram.FakeClient{SentMessage: newSentMessage()}
	h := NewSendReply(&client)
	s := newSessionWithReply(newMessage(), "hi yourself")
	ctx := context.WithValue(context.Background(), key{}, "marker")

	if err := h.Handle(ctx, s); err != nil {
		t.Fatalf("want no error, got %v", err)
	}
	if len(client.Contexts) != 1 {
		t.Fatalf("want 1 context, got %d", len(client.Contexts))
	}
	if got := client.Contexts[0].Value(key{}); got != "marker" {
		t.Errorf("want=%q, got=%v", "marker", got)
	}
}

func TestSendReplyRejectsSessionWithoutMessage(t *testing.T) {
	client := telegram.FakeClient{SentMessage: newSentMessage()}
	h := NewSendReply(&client)
	s := newSessionWithReply(nil, "hi yourself")

	err := h.Handle(context.Background(), s)

	if err == nil {
		t.Fatal("want an error, got nil")
	}
	if !errors.Is(err, ErrSendReply) {
		t.Errorf("want ErrSendReply, got %v", err)
	}
	if got := client.CallCounts["SendMessage"]; got != 0 {
		t.Errorf("want no call, got %d", got)
	}
	if s.SentMessage != nil {
		t.Errorf("want no sent message on the session, got %+v", s.SentMessage)
	}
}

func TestSendReplyRejectsSessionWithoutReply(t *testing.T) {
	client := telegram.FakeClient{SentMessage: newSentMessage()}
	h := NewSendReply(&client)
	s := newSessionWithReply(newMessage(), "")

	err := h.Handle(context.Background(), s)

	if err == nil {
		t.Fatal("want an error, got nil")
	}
	if !errors.Is(err, ErrSendReply) {
		t.Errorf("want ErrSendReply, got %v", err)
	}
	if got := client.CallCounts["SendMessage"]; got != 0 {
		t.Errorf("want no call, got %d", got)
	}
	if s.SentMessage != nil {
		t.Errorf("want no sent message on the session, got %+v", s.SentMessage)
	}
}

func TestSendReplyWrapsClientError(t *testing.T) {
	clientErr := errors.New("connection refused")
	client := telegram.FakeClient{SentMessage: newSentMessage(), Error: clientErr}
	h := NewSendReply(&client)
	s := newSessionWithReply(newMessage(), "hi yourself")

	err := h.Handle(context.Background(), s)

	if err == nil {
		t.Fatal("want an error, got nil")
	}
	if !errors.Is(err, ErrSendReply) {
		t.Errorf("want ErrSendReply, got %v", err)
	}
	if !errors.Is(err, clientErr) {
		t.Errorf("want the client error, got %v", err)
	}
	if s.SentMessage != nil {
		t.Errorf("want no sent message on the session, got %+v", s.SentMessage)
	}
}

func TestNewSendReplyRejectsANilClient(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("want a panic on a nil client, got none")
		}
	}()

	NewSendReply(nil)
}

func TestSendReplySatisfiesHandler(t *testing.T) {
	client := telegram.FakeClient{SentMessage: newSentMessage()}
	var h shared.Handler[*session.Session] = NewSendReply(&client)
	s := newSessionWithReply(newMessage(), "hi yourself")

	if err := h.Handle(context.Background(), s); err != nil {
		t.Fatalf("want no error, got %v", err)
	}
	if got := client.CallCounts["SendMessage"]; got != 1 {
		t.Errorf("want 1 call, got %d", got)
	}
}
