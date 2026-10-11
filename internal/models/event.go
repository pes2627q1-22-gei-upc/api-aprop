package models

import (
	"time"
)

// Event is the resource. Struct tags control the JSON field names.
type Event struct {
	ID    int64  `json:"id"`
	Title string `json:"title"`
	Subtitle string `json:"title"`
	Schedule string `json:"schedule"`
	Tickets string `json:"tickets"`
	URL string `json:"url"`		// URL of the event post
	TicketsURL string `json:"tickets-url"`	// URL for tickets
	Description  string   `json:"description"`
	Free boolean `json:"free"`
	Permanent boolean `json:"permanent"`
	RelatedURLs []string `json:"related-urls"`	// Related URLs for extra info
	URLnames []string `json:"url-names"`	// Name to be used for each related URL
	Online boolean `json:"online"`
	Images []string `json:"images"`
	Videos string `json:"videos"`
	Address string `json:"address"`
	PostalCode string `json:"postal-code"`
	Email string `json:"email"`
	Place string `json:"place"`
	Latitude float `json:"latitude"`
	Longitude float `json:"longitude"`
	Phone float `json:"phone"`
	ActivityURL float `json:"activity-url"`		// URL where the activity is held (for online events)
	StartDate time.Time `json:"start-date"`
	EndDate time.Time `json:"end-date"` 
}