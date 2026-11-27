package rating_model

// Rating is a user's score for an event.
type Rating struct {
	UserID  string `json:"user-id"`
	EventID int    `json:"event-id"`
	Score   int    `json:"score"` // 1..5 (R9)
}

type UserRating struct {
	Rater string `json:"rater"`
	Rated string `json:"rated"`
	Event int    `json:"event"`
	Score int    `json:"score"` // 1..5 (R9)
}

// AttendanceStatus reports whether a user has confirmed attendance at an event.
type AttendanceStatus struct {
	UserID      string `json:"user-id"`
	EventID     int    `json:"event-id"`
	HasAttended bool   `json:"has-attended"`
}
