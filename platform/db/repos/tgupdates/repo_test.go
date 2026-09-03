package tgupdates

import (
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/paveltessman/yaa/pipelines/telegram/updates/models"
	"github.com/paveltessman/yaa/platform/db/dbtest"
	"github.com/paveltessman/yaa/platform/db/internal/sqlc"
)

func newMessage() *models.Message {
	message := models.Message{
		ID:       10,
		ChatID:   40,
		ThreadID: 20,
		UserID:   30,
		Type:     models.FromUser,
		Date:     time.Unix(1700000000, 0),
		Text:     "hello",
	}
	return &message
}

func read(t *testing.T, pool *pgxpool.Pool, chatID, messageID int64) sqlc.TgMessage {
	t.Helper()

	const query = `SELECT id, message_id, chat_id, thread_id, user_id, type, date, text
		FROM tg_message WHERE chat_id = $1 AND message_id = $2`

	var got sqlc.TgMessage
	err := pool.QueryRow(t.Context(), query, chatID, messageID).Scan(
		&got.ID, &got.MessageID, &got.ChatID, &got.ThreadID,
		&got.UserID, &got.Type, &got.Date, &got.Text,
	)
	if err != nil {
		t.Fatalf("reading the row back: %v", err)
	}
	return got
}

func count(t *testing.T, pool *pgxpool.Pool) int {
	t.Helper()

	var got int
	if err := pool.QueryRow(t.Context(), `SELECT count(*) FROM tg_message`).Scan(&got); err != nil {
		t.Fatalf("counting the rows: %v", err)
	}
	return got
}

func TestStoreMessageWritesEveryColumn(t *testing.T) {
	pool := dbtest.NewPool(t)
	repo := New(pool)
	message := newMessage()

	if err := repo.StoreMessage(t.Context(), message); err != nil {
		t.Fatalf("want no error, got %v", err)
	}

	got := read(t, pool, message.ChatID, message.ID)
	if got.MessageID != message.ID {
		t.Errorf("message_id: want=%d, got=%d", message.ID, got.MessageID)
	}
	if got.ChatID != message.ChatID {
		t.Errorf("chat_id: want=%d, got=%d", message.ChatID, got.ChatID)
	}
	if got.ThreadID != message.ThreadID {
		t.Errorf("thread_id: want=%d, got=%d", message.ThreadID, got.ThreadID)
	}
	if got.UserID != message.UserID {
		t.Errorf("user_id: want=%d, got=%d", message.UserID, got.UserID)
	}
	if got.Type != string(message.Type) {
		t.Errorf("type: want=%s, got=%s", message.Type, got.Type)
	}
	if !got.Date.Equal(message.Date) {
		t.Errorf("date: want=%s, got=%s", message.Date, got.Date)
	}
	if got.Text != message.Text {
		t.Errorf("text: want=%q, got=%q", message.Text, got.Text)
	}
	if got.ID == (uuid.UUID{}) {
		t.Error("id: want a uuid, got the zero value")
	}
}

func TestStoreMessageStoresBothTypes(t *testing.T) {
	pool := dbtest.NewPool(t)
	repo := New(pool)

	for i, messageType := range []models.MessageType{models.FromUser, models.ToUser} {
		t.Run(string(messageType), func(t *testing.T) {
			message := newMessage()
			message.ID += int64(i)
			message.Type = messageType

			if err := repo.StoreMessage(t.Context(), message); err != nil {
				t.Fatalf("want no error, got %v", err)
			}
			if got := read(t, pool, message.ChatID, message.ID).Type; got != string(messageType) {
				t.Errorf("type: want=%s, got=%s", messageType, got)
			}
		})
	}
}

func TestStoreMessageStoresEmptyText(t *testing.T) {
	pool := dbtest.NewPool(t)
	repo := New(pool)
	message := newMessage()
	message.Text = ""

	if err := repo.StoreMessage(t.Context(), message); err != nil {
		t.Fatalf("want no error, got %v", err)
	}
	if got := read(t, pool, message.ChatID, message.ID).Text; got != "" {
		t.Errorf("text: want an empty string, got %q", got)
	}
}

func TestStoreMessageIgnoresARepeat(t *testing.T) {
	pool := dbtest.NewPool(t)
	repo := New(pool)
	message := newMessage()

	if err := repo.StoreMessage(t.Context(), message); err != nil {
		t.Fatalf("want no error, got %v", err)
	}
	first := read(t, pool, message.ChatID, message.ID)

	repeat := newMessage()
	repeat.Text = "hello again"
	if err := repo.StoreMessage(t.Context(), repeat); err != nil {
		t.Fatalf("want no error on the repeat, got %v", err)
	}

	if got := count(t, pool); got != 1 {
		t.Errorf("want 1 row, got %d", got)
	}
	second := read(t, pool, message.ChatID, message.ID)
	if second.ID != first.ID {
		t.Errorf("id changed: want=%s, got=%s", first.ID, second.ID)
	}
	if second.Text != message.Text {
		t.Errorf("text changed: want=%q, got=%q", message.Text, second.Text)
	}
}

func TestStoreMessageKeepsTheSameIDInAnotherChat(t *testing.T) {
	pool := dbtest.NewPool(t)
	repo := New(pool)
	message := newMessage()
	other := newMessage()
	other.ChatID = message.ChatID + 1

	if err := repo.StoreMessage(t.Context(), message); err != nil {
		t.Fatalf("want no error, got %v", err)
	}
	if err := repo.StoreMessage(t.Context(), other); err != nil {
		t.Fatalf("want no error for the other chat, got %v", err)
	}

	if got := count(t, pool); got != 2 {
		t.Errorf("want 2 rows, got %d", got)
	}
}

func store(t *testing.T, repo *Repo, messages ...*models.Message) {
	t.Helper()

	for _, message := range messages {
		if err := repo.StoreMessage(t.Context(), message); err != nil {
			t.Fatalf("storing message %d: %v", message.ID, err)
		}
	}
}

func ids(messages []*models.Message) []int64 {
	got := make([]int64, 0, len(messages))
	for _, message := range messages {
		got = append(got, message.ID)
	}
	return got
}

func TestLoadThreadReadsEveryColumn(t *testing.T) {
	pool := dbtest.NewPool(t)
	repo := New(pool)
	message := newMessage()
	store(t, repo, message)

	got, err := repo.LoadThread(t.Context(), message.ChatID, message.ThreadID)

	if err != nil {
		t.Fatalf("want no error, got %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("want 1 message, got %d", len(got))
	}
	first := got[0]
	if first.ID != message.ID {
		t.Errorf("id: want=%d, got=%d", message.ID, first.ID)
	}
	if first.ChatID != message.ChatID {
		t.Errorf("chat id: want=%d, got=%d", message.ChatID, first.ChatID)
	}
	if first.ThreadID != message.ThreadID {
		t.Errorf("thread id: want=%d, got=%d", message.ThreadID, first.ThreadID)
	}
	if first.UserID != message.UserID {
		t.Errorf("user id: want=%d, got=%d", message.UserID, first.UserID)
	}
	if first.Type != message.Type {
		t.Errorf("type: want=%s, got=%s", message.Type, first.Type)
	}
	if !first.Date.Equal(message.Date) {
		t.Errorf("date: want=%s, got=%s", message.Date, first.Date)
	}
	if first.Text != message.Text {
		t.Errorf("text: want=%q, got=%q", message.Text, first.Text)
	}
}

func TestLoadThreadOrdersByDateThenID(t *testing.T) {
	pool := dbtest.NewPool(t)
	repo := New(pool)

	older := newMessage()
	older.ID = 3
	older.Date = time.Unix(1700000000, 0)

	sameSecondLow := newMessage()
	sameSecondLow.ID = 1
	sameSecondLow.Date = time.Unix(1700000060, 0)

	sameSecondHigh := newMessage()
	sameSecondHigh.ID = 2
	sameSecondHigh.Date = time.Unix(1700000060, 0)

	store(t, repo, sameSecondHigh, older, sameSecondLow)

	got, err := repo.LoadThread(t.Context(), older.ChatID, older.ThreadID)

	if err != nil {
		t.Fatalf("want no error, got %v", err)
	}
	want := []int64{older.ID, sameSecondLow.ID, sameSecondHigh.ID}
	if !slices.Equal(ids(got), want) {
		t.Errorf("want=%v, got=%v", want, ids(got))
	}
}

func TestLoadThreadSkipsAnotherThreadOfTheSameChat(t *testing.T) {
	pool := dbtest.NewPool(t)
	repo := New(pool)
	message := newMessage()
	other := newMessage()
	other.ID = message.ID + 1
	other.ThreadID = message.ThreadID + 1
	store(t, repo, message, other)

	got, err := repo.LoadThread(t.Context(), message.ChatID, message.ThreadID)

	if err != nil {
		t.Fatalf("want no error, got %v", err)
	}
	want := []int64{message.ID}
	if !slices.Equal(ids(got), want) {
		t.Errorf("want=%v, got=%v", want, ids(got))
	}
}

func TestLoadThreadSkipsAnotherChat(t *testing.T) {
	pool := dbtest.NewPool(t)
	repo := New(pool)
	message := newMessage()
	other := newMessage()
	other.ChatID = message.ChatID + 1
	store(t, repo, message, other)

	got, err := repo.LoadThread(t.Context(), message.ChatID, message.ThreadID)

	if err != nil {
		t.Fatalf("want no error, got %v", err)
	}
	want := []int64{message.ID}
	if !slices.Equal(ids(got), want) {
		t.Errorf("want=%v, got=%v", want, ids(got))
	}
}

func TestLoadThreadReadsBothTypes(t *testing.T) {
	pool := dbtest.NewPool(t)
	repo := New(pool)
	fromUser := newMessage()
	toUser := newMessage()
	toUser.ID = fromUser.ID + 1
	toUser.Type = models.ToUser
	store(t, repo, fromUser, toUser)

	got, err := repo.LoadThread(t.Context(), fromUser.ChatID, fromUser.ThreadID)

	if err != nil {
		t.Fatalf("want no error, got %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("want 2 messages, got %d", len(got))
	}
	if got[0].Type != models.FromUser {
		t.Errorf("type: want=%s, got=%s", models.FromUser, got[0].Type)
	}
	if got[1].Type != models.ToUser {
		t.Errorf("type: want=%s, got=%s", models.ToUser, got[1].Type)
	}
}

func TestLoadThreadReturnsAnEmptyListForAnUnknownThread(t *testing.T) {
	pool := dbtest.NewPool(t)
	repo := New(pool)
	message := newMessage()
	store(t, repo, message)

	got, err := repo.LoadThread(t.Context(), message.ChatID, message.ThreadID+1)

	if err != nil {
		t.Fatalf("want no error, got %v", err)
	}
	if len(got) != 0 {
		t.Errorf("want no message, got %d", len(got))
	}
}
