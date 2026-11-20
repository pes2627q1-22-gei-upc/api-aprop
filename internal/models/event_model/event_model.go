package event_model

import (
	"github.com/jackc/pgx/v5/pgtype"
	"time"
)

// States of an unofficial event (NoOficial.estat). Only Active events can be modified or cancelled.
// TODO(DP-038): Withdrawn (Retirat) is not set by any endpoint yet. Cancelled is set by its creator (DP-041).
const (
	UnofficialStateActive    = "Active"
	UnofficialStateWithdrawn = "Withdrawn"
	UnofficialStateCancelled = "Cancelled"
)

// Cancellation is the result of POST /events/{id}/cancel.
type Cancellation struct {
	EventID       int    `json:"event-id"`
	State         string `json:"state"`
	NotifiedUsers int64  `json:"notified-users"`
}

// Event is the resource. Struct tags control the JSON field names.
type Event struct {
	ID                 int         `json:"id"`
	Code               int64       `json:"code"` // Code given to a specific event. It is NOT UNIQUE and can be reused if the same event is redone or done in multiple places at once
	IsOfficial         bool        `json:"is-official"`
	Title              string      `json:"title"`
	Subtitle           string      `json:"subtitle"`
	Schedule           string      `json:"schedule"`
	Tickets            string      `json:"tickets"`
	URL                string      `json:"url"`         // URL of the event post
	TicketsURL         string      `json:"tickets-url"` // URL for tickets
	Description        string      `json:"description"`
	Free               pgtype.Bool `json:"free"`
	Permanent          pgtype.Bool `json:"permanent"`
	RelatedURLs        string      `json:"related-urls"` // Related URLs for extra info
	URLnames           string      `json:"url-names"`    // Name to be used for each related URL
	Modality           string      `json:"modality"`
	Images             string      `json:"images"`
	Videos             string      `json:"videos"`
	Email              string      `json:"email"`
	Place              string      `json:"place"`
	Latitude           float32     `json:"latitude"`
	Longitude          float32     `json:"longitude"`
	PostalCode         string      `json:"postal-code"`
	Municipality       string      `json:"municipality"`
	Region             string      `json:"region"`
	Address            string      `json:"address"`
	Phone              string      `json:"phone"`
	ActivityURL        string      `json:"activity-url"` // URL where the activity is held (for online events)
	StartDate          time.Time   `json:"start-date"`
	EndDate            time.Time   `json:"end-date"`
	Categories         []string    `json:"categories"`
	Creator            pgtype.UUID `json:"creator"`
	CreatorName        *string     `json:"creator-name"`
	CreatorReliability *float32    `json:"creator-reliability"`
	Rating             *float32    `json:"rating"`
	DateCreated        time.Time   `json:"date-created"`
}

type Category struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// MapFilter defines the visible map bounding box and optional user ID for friend indicators.
type MapFilter struct {
	MinLat float32
	MaxLat float32
	MinLng float32
	MaxLng float32
	UserID string
}

// MapEvent is an Event enriched with map marker metadata (official status and friends indicator).
type MapEvent struct {
	Event
	IsOfficial bool `json:"is-official"`
	HasFriends bool `json:"has-friends"`
}

// SearchResult represents the result of an event search query with total count.
type SearchResult struct {
	Query    string  `json:"query"`
	Total    int64   `json:"total"`
	PageNum  int     `json:"page-num"`
	PageSize int     `json:"page-size"`
	Events   []Event `json:"events"`
}

type ReceivedRating struct {
	Rating     int       `json:"rating"`
	RatingDate time.Time `json:"rating-date"`
}
