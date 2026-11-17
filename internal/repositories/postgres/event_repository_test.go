package postgres

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"api-aprop/internal/models/event_model"
	"api-aprop/internal/repositories/event_repository"
	"api-aprop/internal/repositories/postgres/sqlc"
)

func TestMapEventRowToModel(t *testing.T) {
	start := time.Date(2026, 10, 25, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 10, 26, 0, 0, 0, 0, time.UTC)

	row := sqlc.ListMapEventsRow{
		ID:           42,
		Title:        "Castellers a la plaça",
		Subtitle:     toText("Diada castellera"),
		Description:  toText("Actuació de colles locals"),
		Schedule:     toText("12:00h"),
		Url:          toText("https://example.cat/event"),
		Tickets:      toText("Gratuït"),
		Ticketsurl:   toText("https://example.cat/tickets"),
		Free:         toPGXBool(true),
		Permanent:    toPGXBool(false),
		Modality:     "Face-to-Face",
		Relatedurls:  toText("https://example.cat/info"),
		Urlnames:     toText("Info"),
		Images:       toText("img1.jpg"),
		Videos:       toText("vid1.mp4"),
		Email:        toText("info@example.cat"),
		Place:        toText("Plaça de la Vila"),
		Latitude:     toPGXFloat(41.3851),
		Longitude:    toPGXFloat(2.1734),
		PostalCode:   toText("08001"),
		Municipality: toText("Barcelona"),
		Region:       toText("Barcelonès"),
		Address:      toText("Plaça de la Vila, 1"),
		Phone:        toText("930000000"),
		Activityurl:  toText("https://example.cat/activity"),
		Startdate:    toDate(start),
		Enddate:      toDate(end),
		IsOfficial:   true,
		HasFriends:   true,
	}

	got := mapEventRowToModel(row, []string{})

	if got.ID != 42 || got.Title != "Castellers a la plaça" {
		t.Errorf("unexpected ID/Title: %+v", got)
	}
	if !got.IsOfficial || !got.HasFriends {
		t.Errorf("expected IsOfficial=true and HasFriends=true, got %+v", got)
	}
	if got.Latitude != 41.3851 || got.Longitude != 2.1734 {
		t.Errorf("unexpected coordinates: (%v, %v)", got.Latitude, got.Longitude)
	}
	if got.Modality != "Face-to-Face" || got.Municipality != "Barcelona" || !got.Free.Bool {
		t.Errorf("unexpected mapped fields: %+v", got)
	}
	if !got.StartDate.Equal(start) || !got.EndDate.Equal(end) {
		t.Errorf("unexpected dates: start=%v end=%v", got.StartDate, got.EndDate)
	}
}

func TestListMapEvents_InvalidUUID(t *testing.T) {
	r := &EventRepository{queries: sqlc.New(nil)}
	_, err := r.ListMapEvents(context.Background(), event_model.MapFilter{
		MinLat: -90,
		MaxLat: 90,
		MinLng: -180,
		MaxLng: 180,
		UserID: "invalid-uuid",
	})
	if err == nil {
		t.Fatal("expected error for invalid UUID, got nil")
	}
}

func TestList_InvalidFilters(t *testing.T) {
	valid := func() map[string]any {
		return map[string]any{"pageSize": 20, "pageNum": 0, "filterCategory": 0}
	}
	tests := []struct {
		name  string
		key   string
		value any
	}{
		{"negative page size", "pageSize", -1},
		{"negative page number", "pageNum", -1},
		{"region without type", "filterRegion", "barcelones"},
		{"start before with a wrong format", "startBefore", "2026-10-27"},
		{"start after with a wrong format", "startAfter", "tomorrow"},
		{"end before with a wrong format", "endBefore", "x"},
		{"end after with a wrong format", "endAfter", "x"},
		{"permanent not a boolean", "filterPermanent", "maybe"},
		{"free not a boolean", "filterFree", "maybe"},
		{"official not a boolean", "filterOfficial", "maybe"},
		{"unknown modality", "filterModality", "Hybrid"},
	}

	// The parameters are checked before querying the database, so no connection is needed
	r := &EventRepository{queries: sqlc.New(nil)}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := valid()
			params[tt.key] = tt.value
			_, err := r.List(params)
			if !errors.Is(err, event_repository.ErrInvalidFilter) {
				t.Fatalf("error = %v, want %v", err, event_repository.ErrInvalidFilter)
			}
		})
	}
}

func TestGetPopular_NegativePage(t *testing.T) {
	r := &EventRepository{queries: sqlc.New(nil)}
	if _, err := r.GetPopular(-1, 0); !errors.Is(err, event_repository.ErrInvalidFilter) {
		t.Fatalf("error = %v, want %v", err, event_repository.ErrInvalidFilter)
	}
}

func TestAppendCategory_SkipsNull(t *testing.T) {
	// An event without categories has a single row with a NULL category
	cats := appendCategory([]string{}, pgtype.Text{})
	cats = appendCategory(cats, toText("Música"))
	if !slices.Equal(cats, []string{"Música"}) {
		t.Errorf("categories = %q, want [Música]", cats)
	}
}

func TestToGetEventRow_KeepsTicketsURL(t *testing.T) {
	row := toGetEventRow(sqlc.SearchEventsWithRelevanceRow{
		Url:        toText("https://example.cat/event"),
		Ticketsurl: toText("https://example.cat/tickets"),
	})
	if row.Url.String != "https://example.cat/event" || row.Ticketsurl.String != "https://example.cat/tickets" {
		t.Errorf("url/tickets url = %q/%q", row.Url.String, row.Ticketsurl.String)
	}
}

func TestGroupActiveEventRows(t *testing.T) {
	if got := groupActiveEventRows(nil); len(got) != 0 {
		t.Fatalf("empty input returned %d items, want 0", len(got))
	}

	rows := []sqlc.ListActiveEventsForRecommendationsRow{
		{ID: 1, Title: "Concert", NumSignups: 12, Category: toText("Música"), Code: pgtype.Int8{Int64: 100, Valid: true}},
		{ID: 1, Title: "Concert", NumSignups: 12, Category: toText("Cultura"), Code: pgtype.Int8{Int64: 100, Valid: true}},
		{ID: 2, Title: "Sense categoria", NumSignups: 3, Category: pgtype.Text{}},
	}
	got := groupActiveEventRows(rows)
	if len(got) != 2 {
		t.Fatalf("got %d events, want 2", len(got))
	}
	if got[0].Event.ID != 1 || got[0].SignUpCount != 12 || !slices.Equal(got[0].Event.Categories, []string{"Música", "Cultura"}) {
		t.Errorf("event[0] = %+v", got[0])
	}
	if !got[0].Event.IsOfficial {
		t.Errorf("event[0] IsOfficial = false, want true")
	}
	if got[1].Event.ID != 2 || got[1].SignUpCount != 3 || len(got[1].Event.Categories) != 0 {
		t.Errorf("event[1] = %+v", got[1])
	}
}
