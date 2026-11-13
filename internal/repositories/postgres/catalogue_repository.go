package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"api-aprop/internal/models/event_model"
	"api-aprop/internal/repositories/catalogue_repository"
	"api-aprop/internal/repositories/postgres/sqlc"
)

// CatalogueRepository searches the events catalogue in Postgres.
type CatalogueRepository struct {
	queries *sqlc.Queries
}

var _ catalogue_repository.Repository = (*CatalogueRepository)(nil)

func NewCatalogueRepository(pool *pgxpool.Pool) *CatalogueRepository {
	return &CatalogueRepository{queries: sqlc.New(pool)}
}

func (r *CatalogueRepository) Search(ctx context.Context, filter catalogue_repository.SearchFilter) ([]event_model.Event, error) {
	params := sqlc.SearchEventsParams{
		Category:     toText(filter.Category),
		Municipality: toText(filter.Municipality),
		MaxResults:   int32(filter.Limit),
	}
	if filter.From != nil {
		params.FromDate = toDate(*filter.From)
	}
	if filter.To != nil {
		params.ToDate = toDate(*filter.To)
	}
	if filter.UserID != "" {
		uid, err := parseUUID(filter.UserID)
		if err != nil {
			return nil, err
		}
		params.UserID = uid
	}
	rows, err := r.queries.SearchEvents(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("search events: %w", err)
	}
	events := make([]event_model.Event, 0, len(rows))
	for _, row := range rows {
		events = append(events, toModel(row, 0))
	}
	return events, nil
}

func (r *CatalogueRepository) ListByIDs(ctx context.Context, ids []int) ([]event_model.Event, error) {
	if len(ids) == 0 {
		return []event_model.Event{}, nil
	}
	ids32 := make([]int32, len(ids))
	for i, id := range ids {
		ids32[i] = int32(id)
	}
	rows, err := r.queries.ListEventsByIDs(ctx, ids32)
	if err != nil {
		return nil, fmt.Errorf("list events by id: %w", err)
	}
	byID := make(map[int]event_model.Event, len(rows))
	for _, row := range rows {
		byID[int(row.ID)] = toModel(row, 0)
	}
	events := make([]event_model.Event, 0, len(ids))
	for _, id := range ids {
		if e, ok := byID[id]; ok {
			events = append(events, e)
		}
	}
	return events, nil
}
