package preference_model

import "time"

// TestQuestion is an event presented in the interest test.
type TestQuestion struct {
	EventID     int      `json:"event-id"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Images      string   `json:"images"`
	Categories  []string `json:"categories"`
}

// TestAnswer is a user's answer to an event in the test.
type TestAnswer struct {
	EventID int  `json:"event-id"`
	Likes   bool `json:"likes"`
}

// SubmitTestRequest is the body sent when submitting the test (full or partial).
type SubmitTestRequest struct {
	Answers []TestAnswer `json:"answers"`
	Partial bool         `json:"partial,omitempty"`
}

// UserPreference is the computed preference weight for a category.
type UserPreference struct {
	CategoryID   int     `json:"category-id"`
	CategoryName string  `json:"category-name"`
	Weight       float32 `json:"weight"`
}

// SubmitTestResponse is returned after submitting the test (full or partial).
type SubmitTestResponse struct {
	UserID        string           `json:"user-id"`
	DateTaken     time.Time        `json:"date-taken"`
	Preferences   []UserPreference `json:"preferences"`
	Partial       bool             `json:"partial,omitempty"`
	AnsweredCount int              `json:"answered-count,omitempty"`
}

// TestStatus reports the user's interest test completion, onboarding prompt
// and partial-progress state.
type TestStatus struct {
	Completed     bool       `json:"completed"`
	Prompted      bool       `json:"prompted"`
	InProgress    bool       `json:"in-progress"`
	AnsweredCount int        `json:"answered-count"`
	DateTaken     *time.Time `json:"date-taken"`
}
