// Package congestion_model holds the contract of the congestion estimate served to Mobilicat.
//
// TODO(DP-030): unlike the rest of the API, these JSON keys are snake_case because that is the
// format agreed with the Mobilicat team.
package congestion_model

import "time"

// Level is the congestion band that matches a congestion factor.
type Level string

const (
	LevelLow    Level = "LOW"
	LevelMedium Level = "MEDIUM"
	LevelHigh   Level = "HIGH"
)

// EstimateRequest is the body of POST /api/v1/congestion/estimate. Clients can send a list of
// locations or a single one in Location.
type EstimateRequest struct {
	Timeframe *Timeframe `json:"timeframe,omitempty"`
	Locations []Location `json:"locations,omitempty"`
	Location  *Location  `json:"location,omitempty"`
}

// Timeframe is the time window to estimate. Missing values default to "now".
type Timeframe struct {
	StartTime *time.Time `json:"start_time,omitempty"`
	EndTime   *time.Time `json:"end_time,omitempty"`
}

// Location is a WGS84 point and the radius around it. Pointers tell a missing value from 0.
type Location struct {
	ID        string   `json:"id"`
	Latitude  *float64 `json:"latitude"`
	Longitude *float64 `json:"longitude"`
	RadiusKm  *float64 `json:"radius_km,omitempty"`
}

// EstimateResponse is the response of POST /api/v1/congestion/estimate.
type EstimateResponse struct {
	Results []Result `json:"results"`
}

// Result is the congestion estimate of one requested location.
type Result struct {
	ID               string  `json:"id"`
	Latitude         float64 `json:"latitude"`
	Longitude        float64 `json:"longitude"`
	CongestionFactor float64 `json:"congestion_factor"`
	Level            Level   `json:"level"`
	EventsCount      int     `json:"events_count"`
}

// AreaStats are the events that take place in an area during a timeframe.
type AreaStats struct {
	EventsCount int
	// ExpectedAttendance is the number of sign ups to those events not marked as absent.
	ExpectedAttendance int
}
