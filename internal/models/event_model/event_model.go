package event_model

import (
	"time"
)

// Event is the resource. Struct tags control the JSON field names.
type Event struct {
	ID          int       `json:"id"`
	Code        int64     `json:"code"` // Code given to a specific event. It is NOT UNIQUE and can be reused if the same event is redone or done in multiple places at once
	Title       string    `json:"title"`
	Subtitle    string    `json:"subtitle"`
	Schedule    string    `json:"schedule"`
	Tickets     string    `json:"tickets"`
	URL         string    `json:"url"`         // URL of the event post
	TicketsURL  string    `json:"tickets-url"` // URL for tickets
	Description string    `json:"description"`
	Free        bool      `json:"free"`
	Permanent   bool      `json:"permanent"`
	RelatedURLs []string  `json:"related-urls"` // Related URLs for extra info
	URLnames    []string  `json:"url-names"`    // Name to be used for each related URL
	Online      bool      `json:"online"`
	Images      []string  `json:"images"`
	Videos      string    `json:"videos"`
	Address     string    `json:"address"`
	PostalCode  string    `json:"postal-code"`
	Email       string    `json:"email"`
	Place       string    `json:"place"`
	Latitude    float32   `json:"latitude"`
	Longitude   float32   `json:"longitude"`
	Phone       string    `json:"phone"`
	ActivityURL string    `json:"activity-url"` // URL where the activity is held (for online events)
	StartDate   time.Time `json:"start-date"`
	EndDate     time.Time `json:"end-date"`
}
