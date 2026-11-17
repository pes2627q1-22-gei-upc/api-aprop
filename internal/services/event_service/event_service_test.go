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
	"api-aprop/internal/models/signup_model"
	"api-aprop/internal/repositories/event_repository"
	"api-aprop/internal/repositories/preference_repository"
	"api-aprop/internal/repositories/signup_repository"
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
	activeEvents []event_repository.ActiveEvent
	activeErr    error
	popularErr   error
	categoryErr  error
}

func (f *fakeEventRepository) EventExists(id int) (bool, error) { return false, nil }

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

func (f *fakeEventRepository) ListActive(_ context.Context) ([]event_repository.ActiveEvent, error) {
	return f.activeEvents, f.activeErr
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
			repo := fakeEventRepository{getEvent: tt.stored, getErr: tt.getErr, updateErr: tt.updateErr}
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

// fakePreferenceRepository implements the preference methods used to recommend events.
type fakePreferenceRepository struct {
	preference_repository.Repository
	prefs        []preference_model.UserPreference
	prefsErr     error
	eventCats    map[int][]int
	eventCatsErr error
	allCats      []event_model.Category
	allCatsErr   error
}

func (f *fakePreferenceRepository) GetUserPreferences(_ context.Context, _ string) ([]preference_model.UserPreference, error) {
	return f.prefs, f.prefsErr
}

func (f *fakePreferenceRepository) GetEventCategories(_ context.Context, _ []int) (map[int][]int, error) {
	return f.eventCats, f.eventCatsErr
}

func (f *fakePreferenceRepository) ListAllCategories(_ context.Context) ([]event_model.Category, error) {
	return f.allCats, f.allCatsErr
}

type fakeSignUpRepository struct {
	signup_repository.Repository
	events  []event_model.Event
	listErr error
}

func (f *fakeSignUpRepository) ListUserEvents(_ context.Context, _ string, _ signup_model.DateRange) ([]event_model.Event, error) {
	return f.events, f.listErr
}

func TestGetRecommendedEvents(t *testing.T) {
	d1 := time.Date(2026, 11, 20, 0, 0, 0, 0, time.UTC)
	d2 := time.Date(2026, 11, 25, 0, 0, 0, 0, time.UTC)

	baseEvents := []event_repository.ActiveEvent{
		{Event: event_model.Event{ID: 1, Title: "Exposició", Categories: []string{"Exposicions"}, EndDate: d2}},
		{Event: event_model.Event{ID: 2, Title: "Concert", Categories: []string{"Música"}, EndDate: d2}},
		{Event: event_model.Event{ID: 3, Title: "Sense categoria", EndDate: d1}},
	}

	t.Run("categories are ordered by preference weight", func(t *testing.T) {
		repo := fakeEventRepository{activeEvents: baseEvents}
		prefRepo := &fakePreferenceRepository{
			prefs: []preference_model.UserPreference{
				{CategoryName: "Música", Weight: 3},
				{CategoryName: "Exposicions", Weight: 0.5},
			},
		}
		svc := NewEventService(&repo, prefRepo, nil, nil).WithSignUpRepository(&fakeSignUpRepository{})
		got, err := svc.GetRecommendedEvents(context.Background(), validUserID, 20, 0)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 3 || got[0].ID != 2 || got[1].ID != 3 || got[2].ID != 1 {
			t.Fatalf("got IDs %v, want [2 3 1]", eventIDs(got))
		}
	})

	t.Run("category without preference has default weight 1", func(t *testing.T) {
		repo := fakeEventRepository{activeEvents: baseEvents}
		prefRepo := &fakePreferenceRepository{
			prefs: []preference_model.UserPreference{{CategoryName: "Música", Weight: 0.5}},
		}
		svc := NewEventService(&repo, prefRepo, nil, nil)
		got, err := svc.GetRecommendedEvents(context.Background(), validUserID, 20, 0)
		if err != nil {
			t.Fatal(err)
		}
		// ID 3 has earlier EndDate (d1 < d2) so ties with ID 1 on weight 1.0 come with 3 before 1, and 2 last (0.5).
		if len(got) != 3 || got[0].ID != 3 || got[1].ID != 1 || got[2].ID != 2 {
			t.Fatalf("got IDs %v, want [3 1 2]", eventIDs(got))
		}
	})

	t.Run("user without preferences or signups gets all active events ordered by popularity then end date", func(t *testing.T) {
		active := []event_repository.ActiveEvent{
			{Event: event_model.Event{ID: 1, Title: "Exposició", Categories: []string{"Exposicions"}, EndDate: d1}, SignUpCount: 0},
			{Event: event_model.Event{ID: 2, Title: "Concert popular", Categories: []string{"Música"}, EndDate: d2}, SignUpCount: 15},
			{Event: event_model.Event{ID: 3, Title: "Sense data"}, SignUpCount: 0},
		}
		repo := fakeEventRepository{activeEvents: active}
		svc := NewEventService(&repo, &fakePreferenceRepository{}, nil, nil).WithSignUpRepository(&fakeSignUpRepository{})
		got, err := svc.GetRecommendedEvents(context.Background(), validUserID, 20, 0)
		if err != nil {
			t.Fatal(err)
		}
		// 2 (popular) -> 1 (d1) -> 3 (no end date, nulls last)
		if len(got) != 3 || got[0].ID != 2 || got[1].ID != 1 || got[2].ID != 3 {
			t.Fatalf("got IDs %v, want [2 1 3]", eventIDs(got))
		}
	})

	t.Run("signups boost matching categories and exclude already enrolled events", func(t *testing.T) {
		active := []event_repository.ActiveEvent{
			{Event: event_model.Event{ID: 1, Title: "Exposició", Categories: []string{"Exposicions"}, EndDate: d1}},
			{Event: event_model.Event{ID: 2, Title: "Concert ja inscrit", Categories: []string{"Música"}, EndDate: d1}},
			{Event: event_model.Event{ID: 4, Title: "Altre concert", Categories: []string{"Música"}, EndDate: d2}},
		}
		repo := fakeEventRepository{activeEvents: active}
		signUpRepo := &fakeSignUpRepository{
			// Postgres ListUserEvents returns events without Categories; service resolves them from activeEvents.
			events: []event_model.Event{{ID: 2, Title: "Concert ja inscrit"}},
		}
		svc := NewEventService(&repo, &fakePreferenceRepository{}, nil, nil).WithSignUpRepository(signUpRepo)
		got, err := svc.GetRecommendedEvents(context.Background(), validUserID, 20, 0)
		if err != nil {
			t.Fatal(err)
		}
		// Event 2 is excluded; event 4 gets +0.5 boost for "Música" so ranks ahead of event 1.
		if len(got) != 2 || got[0].ID != 4 || got[1].ID != 1 {
			t.Fatalf("got IDs %v, want [4 1]", eventIDs(got))
		}
	})

	t.Run("past signups without categories resolve via preference repository and reinforce preference weights", func(t *testing.T) {
		active := []event_repository.ActiveEvent{
			{Event: event_model.Event{ID: 1, Title: "Exposició", Categories: []string{"Exposicions"}, EndDate: d1}},
			{Event: event_model.Event{ID: 2, Title: "Concert", Categories: []string{"Música"}, EndDate: d2}},
		}
		repo := fakeEventRepository{activeEvents: active}
		prefRepo := &fakePreferenceRepository{
			prefs: []preference_model.UserPreference{
				{CategoryID: 1, CategoryName: "Exposicions", Weight: 1.2},
				{CategoryID: 2, CategoryName: "Música", Weight: 0.9},
			},
			eventCats: map[int][]int{99: {2}, 100: {2}},
			allCats:   []event_model.Category{{ID: 1, Name: "Exposicions"}, {ID: 2, Name: "Música"}},
		}
		signUpRepo := &fakeSignUpRepository{
			events: []event_model.Event{{ID: 99}, {ID: 100}},
		}
		svc := NewEventService(&repo, prefRepo, nil, nil).WithSignUpRepository(signUpRepo)
		got, err := svc.GetRecommendedEvents(context.Background(), validUserID, 20, 0)
		if err != nil {
			t.Fatal(err)
		}
		// Música gets 0.9 + 2*0.5 = 1.9 > Exposicions 1.2
		if len(got) != 2 || got[0].ID != 2 || got[1].ID != 1 {
			t.Fatalf("got IDs %v, want [2 1]", eventIDs(got))
		}
	})

	t.Run("global ordering is applied before pagination", func(t *testing.T) {
		// Low-scored events appear first in repository slice; highest-scored appear last.
		active := []event_repository.ActiveEvent{
			{Event: event_model.Event{ID: 10, Title: "Baix 1", Categories: []string{"Exposicions"}, EndDate: d1}},
			{Event: event_model.Event{ID: 20, Title: "Baix 2", Categories: []string{"Exposicions"}, EndDate: d2}},
			{Event: event_model.Event{ID: 30, Title: "Mitjà", Categories: []string{"Teatre"}, EndDate: d1}},
			{Event: event_model.Event{ID: 40, Title: "Alt 1", Categories: []string{"Música"}, EndDate: d1}},
			{Event: event_model.Event{ID: 50, Title: "Alt 2", Categories: []string{"Música"}, EndDate: d2}},
		}
		repo := fakeEventRepository{activeEvents: active}
		prefRepo := &fakePreferenceRepository{
			prefs: []preference_model.UserPreference{
				{CategoryName: "Música", Weight: 3.0},
				{CategoryName: "Teatre", Weight: 2.0},
				{CategoryName: "Exposicions", Weight: 0.5},
			},
		}
		svc := NewEventService(&repo, prefRepo, nil, nil).WithSignUpRepository(&fakeSignUpRepository{})

		page0, err := svc.GetRecommendedEvents(context.Background(), validUserID, 2, 0)
		if err != nil {
			t.Fatal(err)
		}
		if got := eventIDs(page0); len(got) != 2 || got[0] != 40 || got[1] != 50 {
			t.Fatalf("page 0 IDs = %v, want [40 50]", got)
		}

		page1, err := svc.GetRecommendedEvents(context.Background(), validUserID, 2, 1)
		if err != nil {
			t.Fatal(err)
		}
		if got := eventIDs(page1); len(got) != 2 || got[0] != 30 || got[1] != 10 {
			t.Fatalf("page 1 IDs = %v, want [30 10]", got)
		}

		page2, err := svc.GetRecommendedEvents(context.Background(), validUserID, 2, 2)
		if err != nil {
			t.Fatal(err)
		}
		if got := eventIDs(page2); len(got) != 1 || got[0] != 20 {
			t.Fatalf("page 2 IDs = %v, want [20]", got)
		}

		page3, err := svc.GetRecommendedEvents(context.Background(), validUserID, 2, 3)
		if err != nil {
			t.Fatal(err)
		}
		if page3 == nil || len(page3) != 0 {
			t.Fatalf("page 3 = %#v, want empty non-nil slice", page3)
		}
	})

	t.Run("invalid parameters", func(t *testing.T) {
		svc := NewEventService(&fakeEventRepository{}, &fakePreferenceRepository{}, nil, nil)
		cases := []struct {
			name     string
			userID   string
			pageSize int
			pageNum  int
			wantErr  error
		}{
			{"invalid user id", "not-a-uuid", 10, 0, ErrInvalidUserID},
			{"zero page size", validUserID, 0, 0, ErrInvalidFilter},
			{"negative page size", validUserID, -5, 0, ErrInvalidFilter},
			{"negative page num", validUserID, 10, -1, ErrInvalidFilter},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				_, err := svc.GetRecommendedEvents(context.Background(), tc.userID, tc.pageSize, tc.pageNum)
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("error = %v, want %v", err, tc.wantErr)
				}
			})
		}
	})

	t.Run("error propagation", func(t *testing.T) {
		dbErr := errors.New("db error")

		t.Run("event repository ListActive error", func(t *testing.T) {
			svc := NewEventService(&fakeEventRepository{activeErr: dbErr}, &fakePreferenceRepository{}, nil, nil)
			if _, err := svc.GetRecommendedEvents(context.Background(), validUserID, 10, 0); !errors.Is(err, dbErr) {
				t.Fatalf("error = %v, want %v", err, dbErr)
			}
		})

		t.Run("preference repository GetUserPreferences error", func(t *testing.T) {
			svc := NewEventService(&fakeEventRepository{activeEvents: baseEvents}, &fakePreferenceRepository{prefsErr: dbErr}, nil, nil)
			if _, err := svc.GetRecommendedEvents(context.Background(), validUserID, 10, 0); !errors.Is(err, dbErr) {
				t.Fatalf("error = %v, want %v", err, dbErr)
			}
		})

		t.Run("signup repository ListUserEvents error", func(t *testing.T) {
			svc := NewEventService(&fakeEventRepository{activeEvents: baseEvents}, &fakePreferenceRepository{}, nil, nil).
				WithSignUpRepository(&fakeSignUpRepository{listErr: dbErr})
			if _, err := svc.GetRecommendedEvents(context.Background(), validUserID, 10, 0); !errors.Is(err, dbErr) {
				t.Fatalf("error = %v, want %v", err, dbErr)
			}
		})

		t.Run("preference repository GetEventCategories error", func(t *testing.T) {
			svc := NewEventService(&fakeEventRepository{activeEvents: baseEvents}, &fakePreferenceRepository{eventCatsErr: dbErr}, nil, nil).
				WithSignUpRepository(&fakeSignUpRepository{events: []event_model.Event{{ID: 99}}})
			if _, err := svc.GetRecommendedEvents(context.Background(), validUserID, 10, 0); !errors.Is(err, dbErr) {
				t.Fatalf("error = %v, want %v", err, dbErr)
			}
		})

		t.Run("preference repository ListAllCategories error", func(t *testing.T) {
			svc := NewEventService(
				&fakeEventRepository{activeEvents: baseEvents},
				&fakePreferenceRepository{eventCats: map[int][]int{99: {1}}, allCatsErr: dbErr},
				nil,
				nil,
			).WithSignUpRepository(&fakeSignUpRepository{events: []event_model.Event{{ID: 99}}})
			if _, err := svc.GetRecommendedEvents(context.Background(), validUserID, 10, 0); !errors.Is(err, dbErr) {
				t.Fatalf("error = %v, want %v", err, dbErr)
			}
		})
	})
}

func eventIDs(events []event_model.Event) []int {
	ids := make([]int, len(events))
	for i, e := range events {
		ids[i] = e.ID
	}
	return ids
}
