package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/paveltessman/yaa/pipelines/shared"
	"github.com/paveltessman/yaa/pipelines/telegram/updates/session"
	"github.com/paveltessman/yaa/platform/telegram"
)

func newSession(raw string) *session.Session {
	s := session.Session{
		BaseSession: shared.NewSession(),
		RawUpdate:   []byte(raw),
	}
	return &s
}

func TestParseUpdateKeepsMessage(t *testing.T) {
	cases := map[string]struct {
		raw  string
		want telegram.Message
	}{
		"full message": {
			`{"update_id":1,"message":{"message_id":10,"message_thread_id":20,
			  "from":{"id":30},"chat":{"id":40},"text":"hello","date":1700000000}}`,
			telegram.Message{
				ID: 10, ThreadID: 20,
				From: telegram.User{ID: 30},
				Chat: telegram.Chat{ID: 40},
				Text: "hello", Date: 1700000000,
			},
		},
		"message without thread": {
			`{"message":{"message_id":11,"from":{"id":31},"chat":{"id":41},"text":"hi","date":1700000001}}`,
			telegram.Message{
				ID:   11,
				From: telegram.User{ID: 31},
				Chat: telegram.Chat{ID: 41},
				Text: "hi", Date: 1700000001,
			},
		},
		"message without text": {
			`{"message":{"message_id":12,"from":{"id":32},"chat":{"id":42},"date":1700000002}}`,
			telegram.Message{
				ID:   12,
				From: telegram.User{ID: 32},
				Chat: telegram.Chat{ID: 42},
				Date: 1700000002,
			},
		},
		"empty message object": {
			`{"message":{}}`,
			telegram.Message{},
		},
		"unknown fields are dropped": {
			`{"message":{"message_id":13,"sticker":{"emoji":"x"},"date":1700000003},"edited_message":{"message_id":99}}`,
			telegram.Message{ID: 13, Date: 1700000003},
		},
	}
	for name, key := range cases {
		t.Run(name, func(t *testing.T) {
			s := newSession(key.raw)

			if err := ParseUpdate(context.Background(), s); err != nil {
				t.Fatalf("want no error, got %v", err)
			}
			if s.Update == nil {
				t.Fatal("want an update, got nil")
			}
			if s.Update.Message == nil {
				t.Fatal("want a message, got nil")
			}
			if got := *s.Update.Message; got != key.want {
				t.Errorf("want=%+v, got=%+v", key.want, got)
			}
		})
	}
}

func TestParseUpdateDiscardsUpdateWithoutMessage(t *testing.T) {
	cases := map[string]string{
		"no message key":      `{"update_id":1}`,
		"message is null":     `{"update_id":1,"message":null}`,
		"only another update": `{"update_id":1,"edited_message":{"message_id":10}}`,
	}
	for name, raw := range cases {
		t.Run(name, func(t *testing.T) {
			s := newSession(raw)

			if err := ParseUpdate(context.Background(), s); err != nil {
				t.Fatalf("want no error, got %v", err)
			}
			if s.Update != nil {
				t.Errorf("want no update, got %+v", s.Update)
			}
		})
	}
}

func TestParseUpdateRejectsBadJSON(t *testing.T) {
	cases := map[string]string{
		"empty body":          ``,
		"truncated object":    `{"message":`,
		"not an object":       `[1,2,3]`,
		"message is a string": `{"message":"hello"}`,
		"wrong field type":    `{"message":{"message_id":"ten"}}`,
	}
	for name, raw := range cases {
		t.Run(name, func(t *testing.T) {
			s := newSession(raw)

			err := ParseUpdate(context.Background(), s)

			if err == nil {
				t.Fatal("want an error, got nil")
			}
			if s.Update != nil {
				t.Errorf("want no update on error, got %+v", s.Update)
			}
		})
	}
}

func TestParseUpdateErrorType(t *testing.T) {
	s := newSession(`{"message":{"message_id":"ten"}}`)

	err := ParseUpdate(context.Background(), s)

	var want *json.UnmarshalTypeError
	if !errors.As(err, &want) {
		t.Errorf("want *json.UnmarshalTypeError, got %T (%v)", err, err)
	}
}

func TestParseUpdateKeepsRawUpdate(t *testing.T) {
	raw := `{"message":{"message_id":10,"date":1700000000}}`
	s := newSession(raw)

	if err := ParseUpdate(context.Background(), s); err != nil {
		t.Fatalf("want no error, got %v", err)
	}
	if got := string(s.RawUpdate); got != raw {
		t.Errorf("want=%q, got=%q", raw, got)
	}
}

func TestParseUpdateSatisfiesHandler(t *testing.T) {
	var h shared.Handler[*session.Session] = shared.HandlerFunc[*session.Session](ParseUpdate)
	s := newSession(`{"message":{"message_id":10,"date":1700000000}}`)

	if err := h.Handle(context.Background(), s); err != nil {
		t.Fatalf("want no error, got %v", err)
	}
	if s.Update == nil {
		t.Error("want an update, got nil")
	}
}
