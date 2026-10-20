package signup_model

import "time"

// SignUp is the result of signing a user up to an event.
type SignUp struct {
	UserID  string `json:"user-id"`
	EventID int    `json:"event-id"`
	Created bool   `json:"created"` // false when the user was already signed up
}

// DateRange is an optional, inclusive date range. A nil bound means unbounded.
type DateRange struct {
	From *time.Time
	To   *time.Time
}
