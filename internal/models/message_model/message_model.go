package message_model

import "time"

// MaxTextLength is the longest message accepted in an event chat, in characters.
const MaxTextLength = 1000

// Message is a message of an event chat (Missatge) with its author.
type Message struct {
	UserID         string    `json:"user-id"`
	Username       string    `json:"username"`
	Name           string    `json:"name"`
	ProfilePicture string    `json:"profile-picture"`
	EventID        int       `json:"event-id"`
	Text           string    `json:"text"`
	DateSent       time.Time `json:"date-sent"`
}

// MessagePage is a page of GET /events/{id}/messages, newest first. Total counts every message of the event.
type MessagePage struct {
	EventID  int       `json:"event-id"`
	Total    int64     `json:"total"`
	PageNum  int       `json:"page-num"`
	PageSize int       `json:"page-size"`
	Messages []Message `json:"messages"`
}
