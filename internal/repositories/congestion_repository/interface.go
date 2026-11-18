package congestion_repository

import (
	"context"
	"time"

	"api-aprop/internal/models/congestion_model"
)

// Repository reads the events that cause congestion in an area.
type Repository interface {
	// AreaStats counts the in-person events within radiusKm of (lat, lng) whose dates overlap the
	// days [from, to], and their expected attendance. Only the date part of from and to is used.
	AreaStats(ctx context.Context, lat, lng, radiusKm float64, from, to time.Time) (congestion_model.AreaStats, error)
}
