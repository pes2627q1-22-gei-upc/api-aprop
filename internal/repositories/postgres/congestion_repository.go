package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"api-aprop/internal/models/congestion_model"
	"api-aprop/internal/repositories/congestion_repository"
	"api-aprop/internal/repositories/postgres/sqlc"
)

type CongestionRepository struct {
	queries *sqlc.Queries
}

var _ congestion_repository.Repository = (*CongestionRepository)(nil)

func NewCongestionRepository(pool *pgxpool.Pool) *CongestionRepository {
	return &CongestionRepository{queries: sqlc.New(pool)}
}

func (r *CongestionRepository) AreaStats(ctx context.Context, lat, lng, radiusKm float64, from, to time.Time) (congestion_model.AreaStats, error) {
	row, err := r.queries.GetAreaCongestionStats(ctx, sqlc.GetAreaCongestionStatsParams{
		Lat:      lat,
		Lng:      lng,
		RadiusKm: radiusKm,
		FromDate: dateOf(from),
		ToDate:   dateOf(to),
	})
	if err != nil {
		return congestion_model.AreaStats{}, fmt.Errorf("get area congestion stats: %w", err)
	}
	return congestion_model.AreaStats{
		EventsCount:        int(row.EventsCount),
		ExpectedAttendance: int(row.ExpectedAttendance),
	}, nil
}

// dateOf keeps the calendar day of t in its own location, so the time zone does not shift the date.
func dateOf(t time.Time) pgtype.Date {
	y, m, d := t.Date()
	return pgtype.Date{Time: time.Date(y, m, d, 0, 0, 0, 0, time.UTC), Valid: true}
}
