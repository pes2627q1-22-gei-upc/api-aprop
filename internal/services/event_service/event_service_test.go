package event_service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"api-aprop/internal/clients/s3_client"
	"api-aprop/internal/models/event_model"
	"api-aprop/internal/models/preference_model"
	"api-aprop/internal/repositories/event_repository"
	"api-aprop/internal/repositories/preference_repository"
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
	updated      event_model.Event
	updateErr    error
	updateCalls  int
	deleteErr    error
	deleteCalls  int
	listEvents   []event_model.Event
	listErr      error
	popularErr   error
	categoryErr  error
	// state is the state of the unofficial event; empty means Active
	state      string
	stateErr   error
	stateCalls int
}

func (f *fakeEventRepository) EventExists(id int) (bool, error) { return false, nil }

func (f *fakeEventRepository) GetUnofficialState(_ context.Context, _ int) (string, error) {
	f.stateCalls++
	if f.stateErr != nil {
		return "", f.stateErr
	}
	if f.state == "" {
		return event_model.UnofficialStateActive, nil
	}
	return f.state, nil
}

func (f *fakeEventRepository) Create(e event_model.Event, _ bool) (event_model.Event, error) {
	f.created = e
	return e, f.createErr
}

func (f *fakeEventRepository) CreateOrReplace(e event_model.Event) (event_model.Event, error) {
	return e, nil
}

func (f *fakeEventRepository) Update(_ int, e event_model.Event, _ bool) (event_model.Event, error) {
	f.updateCalls++
	f.updated = e
	return e, f.updateErr
}

func (f *fakeEventRepository) Delete(_ context.Context, _ int) error {
	f.deleteCalls++
	return f.deleteErr
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
	return f.listEvents, f.listErr
}

func (f *fakeEventRepository) GetPopular(_ int, _ int) ([]event_model.Event, error) {
	return nil, f.popularErr
}

func (f *fakeEventRepository) GetCategories() ([]event_model.Category, error) {
	return nil, nil
}

func (f *fakeEventRepository) GetCategory(_ int) (event_model.Category, error) {
	return event_model.Category{}, f.categoryErr
}

func (f *fakeEventRepository) ListMapEvents(_ context.Context, filter event_model.MapFilter) ([]event_model.MapEvent, error) {
	f.mapCalls++
	f.gotFilter = filter
	return f.mapEvents, f.mapErr
}

func (f *fakeEventRepository) Search(_ context.Context, query string, pageNum, pageSize int) (event_model.SearchResult, error) {
	return event_model.SearchResult{
		Query:    query,
		Total:    1,
		PageNum:  pageNum,
		PageSize: pageSize,
		Events:   []event_model.Event{{ID: 1, Title: "Concert de Jazz"}},
	}, nil
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

func (m *mockS3Client) DeleteAvatar(_ context.Context, _ string, _ string) error {
	return nil
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
			svc := NewEventService(&repo, nil, nil, &mockS3Client{})
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
			event:   event_model.Event{Title: "Title", Description: "Desc", Categories: []string{"C"}, StartDate: futureStart, EndDate: futureStart.AddDate(0, 0, -1)},
			wantErr: ErrInvalidEventDates,
		},
		{
			name:   "start date today at midnight is allowed",
			userID: validUserID,
			event: event_model.Event{
				Title: "Title", Description: "Desc", Categories: []string{"C"},
				StartDate: day(time.Now().UTC()),
			},
		},
		{
			name:   "end date on the start day but earlier is allowed",
			userID: validUserID,
			event: event_model.Event{
				Title: "Title", Description: "Desc", Categories: []string{"C"},
				StartDate: day(futureStart).Add(20 * time.Hour), EndDate: day(futureStart).Add(10 * time.Hour),
			},
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
			svc := NewEventService(&repo, nil, nil, &mockS3Client{})
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
			svc := NewEventService(&repo, nil, nil, &s3)

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

func TestSearchEvents(t *testing.T) {
	repo := &fakeEventRepository{}
	svc := NewEventService(repo, nil, nil, nil)

	// Empty query returns 0 total
	res, err := svc.SearchEvents(context.Background(), "", 0, 20)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Total != 0 || len(res.Events) != 0 {
		t.Errorf("expected empty search result, got %+v", res)
	}

	// Non-empty query searches repo
	res, err = svc.SearchEvents(context.Background(), "jazz", 0, 20)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Total != 1 || len(res.Events) != 1 {
		t.Errorf("unexpected search result: %+v", res)
	}
}

var errStateRead = errors.New("db down")

func TestUpdateUnofficialEvent(t *testing.T) {
	var creator pgtype.UUID
	if err := creator.Scan(validUserID); err != nil {
		t.Fatal(err)
	}
	createdAt := time.Now().UTC().Add(-48 * time.Hour)
	createdToday := day(time.Now().UTC()).Add(23*time.Hour + 59*time.Minute)
	stored := event_model.Event{ID: 7, Title: "Taller", Creator: creator, DateCreated: createdAt}

	changes := event_model.Event{
		Title:       "Taller de Teatre",
		Description: "Nova descripció",
		Categories:  []string{"Teatre"},
		StartDate:   time.Now().UTC().Add(24 * time.Hour),
	}

	tests := []struct {
		name        string
		stored      event_model.Event
		getErr      error
		state       string
		stateErr    error
		changes     event_model.Event
		updateErr   error
		wantErr     error
		wantUpdates int
	}{
		{
			name:        "valid update keeps creator and creation date",
			stored:      stored,
			changes:     changes,
			wantUpdates: 1,
		},
		{
			name:   "R5 is checked against the creation date",
			stored: stored,
			changes: event_model.Event{
				Title: "Taller", Description: "Desc", Categories: []string{"C"},
				StartDate: time.Now().UTC().Add(-24 * time.Hour),
			},
			wantUpdates: 1,
		},
		{
			name:   "start date before the creation date violates R5",
			stored: stored,
			changes: event_model.Event{
				Title: "Taller", Description: "Desc", Categories: []string{"C"},
				StartDate: createdAt.Add(-24 * time.Hour),
			},
			wantErr: ErrInvalidEventDates,
		},
		{
			// The database returns the start date at midnight, so an event created today keeps being editable
			name: "event created today that starts today",
			stored: event_model.Event{
				ID: 7, Title: "Taller", Creator: creator, DateCreated: createdToday,
			},
			changes: event_model.Event{
				Title: "Taller", Description: "Desc", Categories: []string{"C"},
				StartDate: day(createdToday),
			},
			wantUpdates: 1,
		},
		{
			name:    "blank title",
			stored:  stored,
			changes: event_model.Event{Title: " ", Description: "Desc", Categories: []string{"C"}, StartDate: changes.StartDate},
			wantErr: ErrInvalidEventTitle,
		},
		{
			name:    "official event",
			stored:  event_model.Event{ID: 7, Title: "Festa Major", Code: 1, IsOfficial: true},
			changes: changes,
			wantErr: ErrInvalidEvent,
		},
		{
			name:    "event not found",
			getErr:  event_repository.ErrEventNotFound,
			changes: changes,
			wantErr: ErrEventNotFound,
		},
		{
			name:    "withdrawn event cannot be modified",
			stored:  stored,
			state:   event_model.UnofficialStateWithdrawn,
			changes: changes,
			wantErr: ErrEventNotActive,
		},
		{
			// The state is checked before the fields, so the answer does not depend on the body
			name:   "withdrawn event with invalid changes is still not active",
			stored: stored,
			state:  event_model.UnofficialStateWithdrawn,
			changes: event_model.Event{
				Title: "Taller", Description: "Desc", Categories: []string{"C"},
				StartDate: createdAt.Add(-24 * time.Hour),
			},
			wantErr: ErrEventNotActive,
		},
		{
			name:     "state of an event deleted in the meantime",
			stored:   stored,
			stateErr: event_repository.ErrEventNotFound,
			changes:  changes,
			wantErr:  ErrEventNotFound,
		},
		{
			name:     "error reading the state",
			stored:   stored,
			stateErr: errStateRead,
			changes:  changes,
			wantErr:  errStateRead,
		},
		{
			name:        "event deleted while updating",
			stored:      stored,
			changes:     changes,
			updateErr:   event_repository.ErrEventNotFound,
			wantErr:     ErrEventNotFound,
			wantUpdates: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := fakeEventRepository{getEvent: tt.stored, getErr: tt.getErr, updateErr: tt.updateErr, state: tt.state, stateErr: tt.stateErr}
			svc := NewEventService(&repo, nil, nil, nil)
			updated, err := svc.UpdateUnofficialEvent(context.Background(), 7, tt.changes)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v, want %v", err, tt.wantErr)
			}
			if repo.updateCalls != tt.wantUpdates {
				t.Fatalf("repository updates = %d, want %d", repo.updateCalls, tt.wantUpdates)
			}
			if tt.wantErr != nil {
				return
			}
			if updated.ID != 7 || updated.IsOfficial {
				t.Errorf("updated event id/official = %d/%v, want 7/false", updated.ID, updated.IsOfficial)
			}
			if repo.updated.Creator != creator || !repo.updated.DateCreated.Equal(tt.stored.DateCreated) {
				t.Errorf("creator or creation date changed: %v %v", repo.updated.Creator, repo.updated.DateCreated)
			}
		})
	}
}

func TestUpdateOfficialEventDoesNotReadTheState(t *testing.T) {
	repo := fakeEventRepository{getEvent: event_model.Event{ID: 7, Title: "Festa Major", Code: 1, IsOfficial: true}}
	svc := NewEventService(&repo, nil, nil, nil)

	_, err := svc.UpdateUnofficialEvent(context.Background(), 7, event_model.Event{Title: "x"})
	if !errors.Is(err, ErrInvalidEvent) {
		t.Fatalf("error = %v, want ErrInvalidEvent", err)
	}
	if repo.stateCalls != 0 {
		t.Errorf("state reads = %d, want 0", repo.stateCalls)
	}
}

func TestUpdateUnofficialEventImage(t *testing.T) {
	var creator pgtype.UUID
	if err := creator.Scan(validUserID); err != nil {
		t.Fatal(err)
	}
	stored := event_model.Event{
		ID: 7, Title: "Taller", Creator: creator, DateCreated: time.Now().UTC(),
		Images: "https://s3/events/7/poster.jpg",
	}

	tests := []struct {
		name      string
		images    string
		wantImage string
	}{
		{name: "body without image keeps the uploaded one", images: "", wantImage: stored.Images},
		{name: "body with image replaces it", images: "https://s3/events/7/new.jpg", wantImage: "https://s3/events/7/new.jpg"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := fakeEventRepository{getEvent: stored}
			svc := NewEventService(&repo, nil, nil, nil)
			changes := event_model.Event{
				Title: "Taller", Description: "Desc", Categories: []string{"C"},
				StartDate: time.Now().UTC().Add(24 * time.Hour), Images: tt.images,
			}
			if _, err := svc.UpdateUnofficialEvent(context.Background(), 7, changes); err != nil {
				t.Fatal(err)
			}
			if repo.updated.Images != tt.wantImage {
				t.Errorf("images = %q, want %q", repo.updated.Images, tt.wantImage)
			}
		})
	}
}

func TestDeleteUnofficialEvent(t *testing.T) {
	var creator pgtype.UUID
	if err := creator.Scan(validUserID); err != nil {
		t.Fatal(err)
	}
	stored := event_model.Event{ID: 7, Title: "Taller", Creator: creator, DateCreated: time.Now().UTC()}

	tests := []struct {
		name        string
		stored      event_model.Event
		getErr      error
		deleteErr   error
		wantErr     error
		wantDeletes int
	}{
		{
			name:        "unofficial event is deleted",
			stored:      stored,
			wantDeletes: 1,
		},
		{
			name:    "official event",
			stored:  event_model.Event{ID: 7, Title: "Festa Major", Code: 1, IsOfficial: true},
			wantErr: ErrInvalidEvent,
		},
		{
			name:    "event not found",
			getErr:  event_repository.ErrEventNotFound,
			wantErr: ErrEventNotFound,
		},
		{
			name:        "event already deleted",
			stored:      stored,
			deleteErr:   event_repository.ErrEventNotFound,
			wantErr:     ErrEventNotFound,
			wantDeletes: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := fakeEventRepository{getEvent: tt.stored, getErr: tt.getErr, deleteErr: tt.deleteErr}
			svc := NewEventService(&repo, nil, nil, nil)
			err := svc.DeleteUnofficialEvent(context.Background(), 7)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v, want %v", err, tt.wantErr)
			}
			if repo.deleteCalls != tt.wantDeletes {
				t.Errorf("repository deletes = %d, want %d", repo.deleteCalls, tt.wantDeletes)
			}
		})
	}
}

func TestListErrorsAreTranslated(t *testing.T) {
	invalid := fmt.Errorf("list events: %w: pageNum", event_repository.ErrInvalidFilter)
	repo := fakeEventRepository{
		listErr:     invalid,
		popularErr:  invalid,
		categoryErr: fmt.Errorf("get category 9: %w", event_repository.ErrCategoryNotFound),
	}
	svc := NewEventService(&repo, nil, nil, nil)

	if _, err := svc.ListEvents(map[string]any{}); !errors.Is(err, ErrInvalidFilter) {
		t.Errorf("ListEvents error = %v, want %v", err, ErrInvalidFilter)
	}
	if _, err := svc.GetPopularEvents(-1, 0); !errors.Is(err, ErrInvalidFilter) {
		t.Errorf("GetPopularEvents error = %v, want %v", err, ErrInvalidFilter)
	}
	if _, err := svc.GetCategory(9); !errors.Is(err, ErrCategoryNotFound) {
		t.Errorf("GetCategory error = %v, want %v", err, ErrCategoryNotFound)
	}
}

// fakePreferenceRepository only implements the preferences used to recommend events.
type fakePreferenceRepository struct {
	preference_repository.Repository
	prefs []preference_model.UserPreference
}

func (f *fakePreferenceRepository) GetUserPreferences(_ context.Context, _ string) ([]preference_model.UserPreference, error) {
	return f.prefs, nil
}

func TestGetRecommendedEvents(t *testing.T) {
	events := []event_model.Event{
		{ID: 1, Title: "Exposició", Categories: []string{"Exposicions"}},
		{ID: 2, Title: "Concert", Categories: []string{"Música"}},
		{ID: 3, Title: "Sense categoria"},
	}

	// Events with the same weight have no order, so only the first and last events are checked
	tests := []struct {
		name      string
		prefs     []preference_model.UserPreference
		wantFirst int
		wantLast  int
	}{
		{
			name:      "categories are ordered by weight",
			prefs:     []preference_model.UserPreference{{CategoryName: "Música", Weight: 3}, {CategoryName: "Exposicions", Weight: 0.5}},
			wantFirst: 2,
			wantLast:  1,
		},
		{
			name: "user without preferences gets every event",
		},
		{
			name:     "category without preference has weight 1",
			prefs:    []preference_model.UserPreference{{CategoryName: "Música", Weight: 0.5}},
			wantLast: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := fakeEventRepository{listEvents: events}
			svc := NewEventService(&repo, &fakePreferenceRepository{prefs: tt.prefs}, nil, nil)
			got, err := svc.GetRecommendedEvents(context.Background(), validUserID, 20, 0)
			if err != nil {
				t.Fatal(err)
			}
			if len(got) != len(events) {
				t.Fatalf("got %d events, want %d", len(got), len(events))
			}
			if tt.wantFirst != 0 && got[0].ID != tt.wantFirst {
				t.Errorf("first event = %d, want %d", got[0].ID, tt.wantFirst)
			}
			if tt.wantLast != 0 && got[len(got)-1].ID != tt.wantLast {
				t.Errorf("last event = %d, want %d", got[len(got)-1].ID, tt.wantLast)
			}
		})
	}
}
