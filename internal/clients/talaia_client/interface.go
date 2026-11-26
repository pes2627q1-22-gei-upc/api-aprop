package talaia_client

import (
	"context"
	"time"

	"api-aprop/internal/models/event_model"
)

// Client fetches events from the Talaia API.
type Client interface {
	FetchEvents(ctx context.Context, from time.Time, until time.Time) (map[string]event_model.Event, error)
}
