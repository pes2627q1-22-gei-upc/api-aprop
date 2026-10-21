package event_service

import (
	"context"
	"errors"
	"math"
	"testing"

	"api-aprop/internal/models/event_model"
)

const validUserID = "3f2504e0-4f89-11d3-9a0c-0305e82c3301"

type fakeEventRepository struct {
	mapEvents []event_model.MapEvent
	mapErr    error

	mapCalls  int
	gotFilter event_model.MapFilter
}

func (f *fakeEventRepository) Create(e event_model.Event) (event_model.Event, error) {
	return e, nil
}

func (f *fakeEventRepository) CreateOrReplace(e event_model.Event) (event_model.Event, error) {
	return e, nil
}

func (f *fakeEventRepository) Update(_ int, e event_model.Event) (event_model.Event, error) {
	return e, nil
}

func (f *fakeEventRepository) Get(_ int) (event_model.Event, error) {
	return event_model.Event{}, nil
}

func (f *fakeEventRepository) List(_ map[string]any) ([]event_model.Event, error) {
	return nil, nil
}

func (f *fakeEventRepository) ListMapEvents(_ context.Context, filter event_model.MapFilter) ([]event_model.MapEvent, error) {
	f.mapCalls++
	f.gotFilter = filter
	return f.mapEvents, f.mapErr
}

func TestListMapEvents(t *testing.T) {
	repoErr := errors.New("db down")
	sample := []event_model.MapEvent{
		{
			Event:      event_model.Event{ID: 1, Title: "Concert", Latitude: 41.3874, Longitude: 2.1686},
			IsOfficial: true,
			HasFriends: true,
		},
		{
			Event:      event_model.Event{ID: 2, Title: "Trobada", Latitude: 41.3902, Longitude: 2.1540},
			IsOfficial: false,
			HasFriends: false,
		},
	}

	tests := []struct {
		name       string
		filter     event_model.MapFilter
		repo       fakeEventRepository
		wantErr    error
		wantCalls  int
		wantFilter event_model.MapFilter
		want       []event_model.MapEvent
	}{
		{
			name:       "valid bounding box without user",
			filter:     event_model.MapFilter{MinLat: 41.30, MaxLat: 41.50, MinLng: 2.05, MaxLng: 2.25},
			repo:       fakeEventRepository{mapEvents: sample},
			wantCalls:  1,
			wantFilter: event_model.MapFilter{MinLat: 41.30, MaxLat: 41.50, MinLng: 2.05, MaxLng: 2.25},
			want:       sample,
		},
		{
			name:       "valid bounding box with uppercase user id normalizes uuid",
			filter:     event_model.MapFilter{MinLat: -90, MaxLat: 90, MinLng: -180, MaxLng: 180, UserID: "3F2504E0-4F89-11D3-9A0C-0305E82C3301"},
			repo:       fakeEventRepository{mapEvents: sample},
			wantCalls:  1,
			wantFilter: event_model.MapFilter{MinLat: -90, MaxLat: 90, MinLng: -180, MaxLng: 180, UserID: validUserID},
			want:       sample,
		},
		{
			name:       "nil repository slice returns non-nil empty slice",
			filter:     event_model.MapFilter{MinLat: -90, MaxLat: 90, MinLng: -180, MaxLng: 180},
			repo:       fakeEventRepository{mapEvents: nil},
			wantCalls:  1,
			wantFilter: event_model.MapFilter{MinLat: -90, MaxLat: 90, MinLng: -180, MaxLng: 180},
			want:       []event_model.MapEvent{},
		},
		{
			name:    "minLat greater than maxLat",
			filter:  event_model.MapFilter{MinLat: 42.0, MaxLat: 41.0, MinLng: 2.0, MaxLng: 3.0},
			wantErr: ErrInvalidBoundingBox,
		},
		{
			name:    "minLng greater than maxLng",
			filter:  event_model.MapFilter{MinLat: 41.0, MaxLat: 42.0, MinLng: 3.0, MaxLng: 2.0},
			wantErr: ErrInvalidBoundingBox,
		},
		{
			name:    "latitude out of range",
			filter:  event_model.MapFilter{MinLat: -91.0, MaxLat: 42.0, MinLng: 2.0, MaxLng: 3.0},
			wantErr: ErrInvalidBoundingBox,
		},
		{
			name:    "longitude out of range",
			filter:  event_model.MapFilter{MinLat: 41.0, MaxLat: 42.0, MinLng: 2.0, MaxLng: 181.0},
			wantErr: ErrInvalidBoundingBox,
		},
		{
			name:    "NaN coordinate",
			filter:  event_model.MapFilter{MinLat: float32(math.NaN()), MaxLat: 42.0, MinLng: 2.0, MaxLng: 3.0},
			wantErr: ErrInvalidBoundingBox,
		},
		{
			name:    "invalid user id",
			filter:  event_model.MapFilter{MinLat: 41.0, MaxLat: 42.0, MinLng: 2.0, MaxLng: 3.0, UserID: "not-a-uuid"},
			wantErr: ErrInvalidUserID,
		},
		{
			name:       "repository error",
			filter:     event_model.MapFilter{MinLat: 41.0, MaxLat: 42.0, MinLng: 2.0, MaxLng: 3.0},
			repo:       fakeEventRepository{mapErr: repoErr},
			wantErr:    repoErr,
			wantCalls:  1,
			wantFilter: event_model.MapFilter{MinLat: 41.0, MaxLat: 42.0, MinLng: 2.0, MaxLng: 3.0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := tt.repo
			s := NewEventService(&repo, nil)

			got, err := s.ListMapEvents(context.Background(), tt.filter)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v, want %v", err, tt.wantErr)
			}
			if repo.mapCalls != tt.wantCalls {
				t.Errorf("ListMapEvents calls = %d, want %d", repo.mapCalls, tt.wantCalls)
			}
			if tt.wantErr != nil {
				return
			}
			if repo.gotFilter != tt.wantFilter {
				t.Errorf("filter passed to repository = %+v, want %+v", repo.gotFilter, tt.wantFilter)
			}
			if got == nil {
				t.Fatal("events is nil, want non-nil slice")
			}
			if len(got) != len(tt.want) {
				t.Fatalf("got %d events, want %d", len(got), len(tt.want))
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("event[%d] = %+v, want %+v", i, got[i], tt.want[i])
				}
			}
		})
	}
}
