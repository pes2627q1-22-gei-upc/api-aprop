package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"api-aprop/internal/models/event_model"
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

	got := mapEventRowToModel(row)

	if got.ID != 42 || got.Title != "Castellers a la plaça" {
		t.Errorf("unexpected ID/Title: %+v", got)
	}
	if !got.IsOfficial || !got.HasFriends {
		t.Errorf("expected IsOfficial=true and HasFriends=true, got %+v", got)
	}
	if got.Latitude != 41.3851 || got.Longitude != 2.1734 {
		t.Errorf("unexpected coordinates: (%v, %v)", got.Latitude, got.Longitude)
	}
	if got.Modality != "Face-to-Face" || got.Municipality != "Barcelona" || !got.Free {
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

func TestToUUID_ValidAndInvalid(t *testing.T) {
	uid, err := toUUID("3f2504e0-4f89-11d3-9a0c-0305e82c3301")
	if err != nil || !uid.Valid {
		t.Fatalf("expected valid pgtype.UUID, got valid=%v err=%v", uid.Valid, err)
	}

	var empty pgtype.UUID
	if empty.Valid {
		t.Fatal("expected zero pgtype.UUID to be invalid (NULL)")
	}
}
