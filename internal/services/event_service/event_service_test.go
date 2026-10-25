package event_service

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"api-aprop/internal/clients/s3_client"
	"api-aprop/internal/models/event_model"
	"api-aprop/internal/repositories/event_repository"
)

const validUserID = "3f2504e0-4f89-11d3-9a0c-0305e82c3301"

type fakeEventRepository struct {
	mapEvents    []event_model.MapEvent
	mapErr       error
	mapCalls     int
	gotFilter    event_model.MapFilter
	created      event_model.Event
	createErr    error
	getEvent     event_model.Event
	getErr       error
	updatedImg   string
	updateImgErr error
}

func (f *fakeEventRepository) Create(e event_model.Event, _ bool) (event_model.Event, error) {
	f.created = e
	return e, f.createErr
}

func (f *fakeEventRepository) CreateOrReplace(e event_model.Event) (event_model.Event, error) {
	return e, nil
}

func (f *fakeEventRepository) Update(_ int, e event_model.Event, _ bool) (event_model.Event, error) {
	return e, nil
}

func (f *fakeEventRepository) UpdateImage(_ context.Context, _ int, img string) error {
	f.updatedImg = img
	return f.updateImgErr
}

func (f *fakeEventRepository) SetCategories(_ int, _ []string, _ bool) error {
	return nil
}

func (f *fakeEventRepository) Get(_ int) (event_model.Event, error) {
	return f.getEvent, f.getErr
}

func (f *fakeEventRepository) List(_ map[string]any) ([]event_model.Event, error) {
	return nil, nil
}

func (f *fakeEventRepository) GetCategories() ([]event_model.Category, error) {
	return nil, nil
}

func (f *fakeEventRepository) GetCategory(_ int) (event_model.Category, error) {
	return event_model.Category{}, nil
}

func (f *fakeEventRepository) ListMapEvents(_ context.Context, filter event_model.MapFilter) ([]event_model.MapEvent, error) {
	f.mapCalls++
	f.gotFilter = filter
	return f.mapEvents, f.mapErr
}

type mockS3Client struct {
	avatarURL string
	avatarErr error
	imageURL  string
	imageErr  error
}

func (m *mockS3Client) UploadAvatar(_ context.Context, _ string, _ string, _ string, _ io.Reader) (string, error) {
	return m.avatarURL, m.avatarErr
}

func (m *mockS3Client) UploadEventImage(_ context.Context, _ int, _ string, _ string, _ io.Reader) (string, error) {
	return m.imageURL, m.imageErr
}

func TestListMapEvents(t *testing.T) {
	_ = errors.New("db down")
	sample := []event_model.MapEvent{
		{
			Event:      event_model.Event{ID: 1, Title: "Concert", Latitude: 41.3874, Longitude: 2.1686},
			IsOfficial: true,
			HasFriends: true,
		},
	}

	tests := []struct {
		name      string
		filter    event_model.MapFilter
		repo      fakeEventRepository
		wantErr   error
		wantCalls int
	}{
		{
			name:      "valid box",
			filter:    event_model.MapFilter{MinLat: 41.0, MaxLat: 42.0, MinLng: 2.0, MaxLng: 3.0, UserID: validUserID},
			repo:      fakeEventRepository{mapEvents: sample},
			wantCalls: 1,
		},
		{
			name:      "invalid user id",
			filter:    event_model.MapFilter{MinLat: 41.0, MaxLat: 42.0, MinLng: 2.0, MaxLng: 3.0, UserID: "bad"},
			wantErr:   ErrInvalidUserID,
			wantCalls: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := tt.repo
			svc := NewEventService(&repo, nil, &mockS3Client{})
			got, err := svc.ListMapEvents(context.Background(), tt.filter)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v, want %v", err, tt.wantErr)
			}
			if repo.mapCalls != tt.wantCalls {
				t.Errorf("calls = %d, want %d", repo.mapCalls, tt.wantCalls)
			}
			if tt.wantErr == nil && len(got) != len(sample) {
				t.Errorf("got %d events, want %d", len(got), len(sample))
			}
		})
	}
}

func TestCreateUnofficialEvent(t *testing.T) {
	futureStart := time.Now().UTC().Add(24 * time.Hour)
	futureEnd := futureStart.Add(2 * time.Hour)
	pastStart := time.Now().UTC().Add(-24 * time.Hour)

	validEvent := event_model.Event{
		Title:       "Festa Major Popular",
		Description: "Activitats culturals i musicals",
		Categories:  []string{"Musica", "Festes"},
		StartDate:   futureStart,
		EndDate:     futureEnd,
		Latitude:    41.3851,
		Longitude:   2.1734,
	}

	tests := []struct {
		name    string
		userID  string
		event   event_model.Event
		wantErr error
	}{
		{
			name:   "valid unofficial event",
			userID: validUserID,
			event:  validEvent,
		},
		{
			name:    "blank title",
			userID:  validUserID,
			event:   event_model.Event{Title: "  ", Description: "Desc", Categories: []string{"C"}, StartDate: futureStart},
			wantErr: ErrInvalidEventTitle,
		},
		{
			name:    "blank description",
			userID:  validUserID,
			event:   event_model.Event{Title: "Title", Description: "  ", Categories: []string{"C"}, StartDate: futureStart},
			wantErr: ErrInvalidEventDescription,
		},
		{
			name:    "no categories",
			userID:  validUserID,
			event:   event_model.Event{Title: "Title", Description: "Desc", Categories: []string{}, StartDate: futureStart},
			wantErr: ErrInvalidEventCategory,
		},
		{
			name:    "past start date violates R5",
			userID:  validUserID,
			event:   event_model.Event{Title: "Title", Description: "Desc", Categories: []string{"C"}, StartDate: pastStart},
			wantErr: ErrInvalidEventDates,
		},
		{
			name:    "end date before start date violates R5",
			userID:  validUserID,
			event:   event_model.Event{Title: "Title", Description: "Desc", Categories: []string{"C"}, StartDate: futureStart, EndDate: futureStart.Add(-1 * time.Hour)},
			wantErr: ErrInvalidEventDates,
		},
		{
			name:    "invalid coordinates",
			userID:  validUserID,
			event:   event_model.Event{Title: "Title", Description: "Desc", Categories: []string{"C"}, StartDate: futureStart, Latitude: 120},
			wantErr: ErrInvalidCoordinates,
		},
		{
			name:    "invalid user id",
			userID:  "not-uuid",
			event:   validEvent,
			wantErr: ErrInvalidUserID,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := fakeEventRepository{}
			svc := NewEventService(&repo, nil, &mockS3Client{})
			created, err := svc.CreateUnofficialEvent(context.Background(), tt.userID, tt.event)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr != nil {
				return
			}
			if created.IsOfficial {
				t.Errorf("created event IsOfficial = true, want false")
			}
			if !created.Creator.Valid {
				t.Errorf("created event Creator.Valid = false, want true")
			}
			if created.DateCreated.IsZero() {
				t.Errorf("created event DateCreated is zero")
			}
		})
	}
}

func TestUploadEventImage(t *testing.T) {
	tests := []struct {
		name        string
		eventID     int
		filename    string
		contentType string
		repo        fakeEventRepository
		s3Client    mockS3Client
		wantErr     error
		wantURL     string
	}{
		{
			name:        "successful poster upload",
			eventID:     10,
			filename:    "cartell.jpg",
			contentType: "image/jpeg",
			repo:        fakeEventRepository{getEvent: event_model.Event{ID: 10, Title: "Concert"}},
			s3Client:    mockS3Client{imageURL: "https://s3/events/10/poster.jpg"},
			wantURL:     "https://s3/events/10/poster.jpg",
		},
		{
			name:        "event not found",
			eventID:     99,
			filename:    "cartell.jpg",
			contentType: "image/jpeg",
			repo:        fakeEventRepository{getErr: event_repository.ErrEventNotFound},
			wantErr:     ErrEventNotFound,
		},
		{
			name:        "invalid content type",
			eventID:     10,
			filename:    "doc.pdf",
			contentType: "application/pdf",
			repo:        fakeEventRepository{getEvent: event_model.Event{ID: 10}},
			s3Client:    mockS3Client{imageErr: s3_client.ErrInvalidContentType},
			wantErr:     ErrInvalidContentType,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := tt.repo
			s3 := tt.s3Client
			svc := NewEventService(&repo, nil, &s3)

			url, err := svc.UploadEventImage(context.Background(), tt.eventID, tt.filename, tt.contentType, strings.NewReader("image-bytes"))
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr != nil {
				return
			}
			if url != tt.wantURL {
				t.Errorf("url = %q, want %q", url, tt.wantURL)
			}
			if repo.updatedImg != tt.wantURL {
				t.Errorf("repo updatedImg = %q, want %q", repo.updatedImg, tt.wantURL)
			}
		})
	}
}
