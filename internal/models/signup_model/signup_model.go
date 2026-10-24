package signup_model

import "time"

// SignUp is the result of signing a user up to an event.
type SignUp struct {
	UserID  string `json:"user-id"`
	EventID int    `json:"event-id"`
	Created bool   `json:"created"` // false when the user was already signed up
}

// Unenroll is the result of removing a user's sign up from an event.
type Unenroll struct {
	UserID  string `json:"user-id"`
	EventID int    `json:"event-id"`
	Deleted bool   `json:"deleted"` // false when the user was not enrolled
}

// EnrollmentStatus represents the current enrollment state and total attendees of an event.
type EnrollmentStatus struct {
	EventID       int  `json:"event-id"`
	IsEnrolled    bool `json:"is-enrolled"`
	EnrolledCount int  `json:"enrolled-count"`
}

// DateRange is an optional, inclusive date range. A nil bound means unbounded.
type DateRange struct {
	From *time.Time
	To   *time.Time
}
