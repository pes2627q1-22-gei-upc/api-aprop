package rating_model

// Rating is a user's score for an event.
type Rating struct {
	UserID  string `json:"user-id"`
	EventID int    `json:"event-id"`
	Score   int    `json:"score"` // 1..5 (R9)
}
