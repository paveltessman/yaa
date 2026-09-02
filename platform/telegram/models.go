package telegram

import (
	"fmt"
	"time"
)

type Update struct {
	Message *Message
}

type Message struct {
	ID       int64 `json:"message_id"`
	ThreadID int64 `json:"message_thread_id"`
	From     User
	Chat     Chat
	Text     string
	Date     int64
}

func (m *Message) Time() time.Time {
	return time.Unix(m.Date, 0)
}

func (m *Message) String() string {
	s := fmt.Sprintf(
		"Message{ID: %d; ThreadID: %d; From: %s; Chat: %s; Text: %q; Date: %s}",
		m.ID, m.ThreadID, m.From.String(), m.Chat.String(), m.Text, m.Time(),
	)
	return s
}

type User struct {
	ID int64 `json:"id"`
}

func (u *User) String() string {
	return fmt.Sprintf("User{ID: %d}", u.ID)
}

type Chat struct {
	ID int64 `json:"id"`
}

func (c *Chat) String() string {
	return fmt.Sprintf("Chat{ID: %d}", c.ID)
}
