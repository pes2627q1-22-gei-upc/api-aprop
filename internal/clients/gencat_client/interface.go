package gencat_client

import (
	"context"

	"api-aprop/internal/models/event_model"
)

// Client fetches events from the Generalitat de Catalunya open data API.
type Client interface {
	FetchEvents(ctx context.Context, pageNumber int, pageSize int) ([]event_model.Event, error)
}
