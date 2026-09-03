package models

import "time"

type Role string

const (
	Assistant Role = "assistant"
	User      Role = "user"
)

type Message struct {
	Role Role
	Date time.Time
	Text string
}
