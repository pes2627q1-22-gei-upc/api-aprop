package congestion_service

import (
	"context"

	"api-aprop/internal/models/congestion_model"
)

// Service estimates how congested an area is from the events in it and their expected attendance.
type Service interface {
	Estimate(ctx context.Context, req congestion_model.EstimateRequest) (congestion_model.EstimateResponse, error)
}
