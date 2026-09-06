package models

import (
	"fmt"
	"time"
)

type WhUpdate struct {
	Message *WhMessage
}

type WhMessage struct {
	ID       int64 `json:"message_id"`
	ThreadID int64 `json:"message_thread_id"`
	From     WhUser
	Chat     WhChat
	Text     string
	Date     int64
}

func (m *WhMessage) Time() time.Time {
	return time.Unix(m.Date, 0)
}

func (m *WhMessage) String() string {
	s := fmt.Sprintf(
		"WhMessage{ID: %d; ThreadID: %d; From: %s; Chat: %s; Text: %q; Date: %s}",
		m.ID, m.ThreadID, m.From.String(), m.Chat.String(), m.Text, m.Time(),
	)
	return s
}

func (m *WhMessage) ToMessage() *Message {
	message := Message{
		ID:       m.ID,
		ChatID:   m.Chat.ID,
		ThreadID: m.ThreadID,
		UserID:   m.From.ID,
		Type:     FromUser,
		Date:     m.Time(),
		Text:     m.Text,
	}
	return &message
}

type WhUser struct {
	ID int64 `json:"id"`
}

func (u *WhUser) String() string {
	return fmt.Sprintf("User{ID: %d}", u.ID)
}

type WhChat struct {
	ID int64 `json:"id"`
}

func (c *WhChat) String() string {
	return fmt.Sprintf("Chat{ID: %d}", c.ID)
}
