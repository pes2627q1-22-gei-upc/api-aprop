package rating_service

import (
	"context"
	"errors"
	"testing"

	"api-aprop/internal/repositories/rating_repository"
)

const validUserID = "3f2504e0-4f89-11d3-9a0c-0305e82c3301"

type fakeEvents struct{ exists bool }

func (f fakeEvents) EventExists(int) (bool, error) { return f.exists, nil }

type fakeRatings struct {
	attended    bool
	attendErr   error
	created     bool
	createErr   error
	createCalls int
	gotScore    int
	updated     bool
	deleted     bool
	writeErr    error
	writeCalls  int
}

func (f *fakeRatings) Create(_ context.Context, _ string, _ int, score int) (bool, error) {
	f.createCalls++
	f.gotScore = score
	return f.created, f.createErr
}

func (f *fakeRatings) Update(_ context.Context, _ string, _ int, score int) (bool, error) {
	f.writeCalls++
	f.gotScore = score
	return f.updated, f.writeErr
}

func (f *fakeRatings) Delete(context.Context, string, int) (bool, error) {
	f.writeCalls++
	return f.deleted, f.writeErr
}

func (f *fakeRatings) HasAttended(context.Context, string, int) (bool, error) {
	return f.attended, f.attendErr
}

func TestR9_ScoreMustBeBetween1And5(t *testing.T) {
	for _, score := range []int{-1, 0, 6, 100} {
		repo := &fakeRatings{attended: true, created: true}
		_, err := NewRatingService(repo, fakeEvents{true}).Rate(context.Background(), validUserID, 7, score)
		if !errors.Is(err, ErrInvalidScore) || repo.createCalls != 0 {
			t.Errorf("score %d: err = %v calls = %d, want ErrInvalidScore and no insert", score, err, repo.createCalls)
		}
	}
	for score := 1; score <= 5; score++ {
		repo := &fakeRatings{attended: true, created: true}
		r, err := NewRatingService(repo, fakeEvents{true}).Rate(context.Background(), validUserID, 7, score)
		if err != nil || r.Score != score || repo.gotScore != score {
			t.Errorf("score %d: rating = %+v err = %v", score, r, err)
		}
	}
}

func TestR10_RatingRequiresAttendance(t *testing.T) {
	repo := &fakeRatings{attended: false, created: true}
	_, err := NewRatingService(repo, fakeEvents{true}).Rate(context.Background(), validUserID, 7, 4)
	if !errors.Is(err, ErrNotAttended) || repo.createCalls != 0 {
		t.Fatalf("err = %v calls = %d, want ErrNotAttended and no insert", err, repo.createCalls)
	}
}

func TestR8_RateUnknownEvent(t *testing.T) {
	// R8 only lets attendance be set once the event started, so rating needs no extra date check;
	// here the event must exist.
	_, err := NewRatingService(&fakeRatings{attended: true, created: true}, fakeEvents{false}).Rate(context.Background(), validUserID, 7, 4)
	if !errors.Is(err, ErrEventNotFound) {
		t.Fatalf("err = %v, want ErrEventNotFound", err)
	}
}

func TestRate_Duplicate(t *testing.T) {
	_, err := NewRatingService(&fakeRatings{attended: true, created: false}, fakeEvents{true}).Rate(context.Background(), validUserID, 7, 4)
	if !errors.Is(err, ErrAlreadyRated) {
		t.Fatalf("err = %v, want ErrAlreadyRated", err)
	}
}

func TestRate_EventDeletedBetweenChecks(t *testing.T) {
	repo := &fakeRatings{attended: true, createErr: rating_repository.ErrEventNotFound}
	_, err := NewRatingService(repo, fakeEvents{true}).Rate(context.Background(), validUserID, 7, 4)
	if !errors.Is(err, ErrEventNotFound) {
		t.Fatalf("err = %v, want ErrEventNotFound", err)
	}
}

func TestRate_InvalidIDs(t *testing.T) {
	s := NewRatingService(&fakeRatings{}, fakeEvents{true})
	if _, err := s.Rate(context.Background(), "nope", 7, 4); !errors.Is(err, ErrInvalidUserID) {
		t.Errorf("user: err = %v", err)
	}
	if _, err := s.Rate(context.Background(), validUserID, 0, 4); !errors.Is(err, ErrInvalidEventID) {
		t.Errorf("event: err = %v", err)
	}
}

func TestHasAttended(t *testing.T) {
	for _, want := range []bool{true, false} {
		status, err := NewRatingService(&fakeRatings{attended: want}, fakeEvents{true}).HasAttended(context.Background(), validUserID, 7)
		if err != nil || status.UserID != validUserID || status.EventID != 7 || status.HasAttended != want {
			t.Errorf("want=%v: got %+v err=%v", want, status, err)
		}
	}

	if _, err := NewRatingService(&fakeRatings{}, fakeEvents{false}).HasAttended(context.Background(), validUserID, 7); !errors.Is(err, ErrEventNotFound) {
		t.Errorf("unknown event: err = %v, want ErrEventNotFound", err)
	}
	if _, err := NewRatingService(&fakeRatings{}, fakeEvents{true}).HasAttended(context.Background(), "nope", 7); !errors.Is(err, ErrInvalidUserID) {
		t.Errorf("invalid user: err = %v, want ErrInvalidUserID", err)
	}
	if _, err := NewRatingService(&fakeRatings{}, fakeEvents{true}).HasAttended(context.Background(), validUserID, 0); !errors.Is(err, ErrInvalidEventID) {
		t.Errorf("invalid event: err = %v, want ErrInvalidEventID", err)
	}
	wantErr := errors.New("db error")
	if _, err := NewRatingService(&fakeRatings{attendErr: wantErr}, fakeEvents{true}).HasAttended(context.Background(), validUserID, 7); !errors.Is(err, wantErr) {
		t.Errorf("repo error: err = %v, want %v", err, wantErr)
	}
}

func TestUpdateRating(t *testing.T) {
	ctx := context.Background()
	dbErr := errors.New("db error")
	tests := []struct {
		name      string
		repo      *fakeRatings
		events    bool
		userID    string
		eventID   int
		score     int
		wantErr   error
		wantCalls int
	}{
		{"ok", &fakeRatings{updated: true}, true, validUserID, 7, 4, nil, 1},
		{"R9 score 0", &fakeRatings{updated: true}, true, validUserID, 7, 0, ErrInvalidScore, 0},
		{"R9 score 6", &fakeRatings{updated: true}, true, validUserID, 7, 6, ErrInvalidScore, 0},
		{"invalid user", &fakeRatings{updated: true}, true, "nope", 7, 3, ErrInvalidUserID, 0},
		{"invalid event", &fakeRatings{updated: true}, true, validUserID, 0, 3, ErrInvalidEventID, 0},
		{"unknown event", &fakeRatings{updated: true}, false, validUserID, 7, 3, ErrEventNotFound, 0},
		{"no rating yet", &fakeRatings{}, true, validUserID, 7, 3, ErrRatingNotFound, 1},
		{"repo error", &fakeRatings{writeErr: dbErr}, true, validUserID, 7, 3, dbErr, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewRatingService(tt.repo, fakeEvents{tt.events}).UpdateRating(ctx, tt.userID, tt.eventID, tt.score)
			if !errors.Is(err, tt.wantErr) || tt.repo.writeCalls != tt.wantCalls {
				t.Fatalf("err = %v calls = %d, want %v / %d", err, tt.repo.writeCalls, tt.wantErr, tt.wantCalls)
			}
			if err == nil && (got.Score != tt.score || got.EventID != tt.eventID || got.UserID != validUserID || tt.repo.gotScore != tt.score) {
				t.Fatalf("rating = %+v", got)
			}
		})
	}
}

func TestDeleteRating(t *testing.T) {
	ctx := context.Background()
	dbErr := errors.New("db error")
	tests := []struct {
		name    string
		repo    *fakeRatings
		events  bool
		userID  string
		eventID int
		wantErr error
	}{
		{"ok", &fakeRatings{deleted: true}, true, validUserID, 7, nil},
		{"invalid user", &fakeRatings{deleted: true}, true, "nope", 7, ErrInvalidUserID},
		{"invalid event", &fakeRatings{deleted: true}, true, validUserID, -1, ErrInvalidEventID},
		{"unknown event", &fakeRatings{deleted: true}, false, validUserID, 7, ErrEventNotFound},
		{"no rating", &fakeRatings{}, true, validUserID, 7, ErrRatingNotFound},
		{"repo error", &fakeRatings{writeErr: dbErr}, true, validUserID, 7, dbErr},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewRatingService(tt.repo, fakeEvents{tt.events}).DeleteRating(ctx, tt.userID, tt.eventID)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}
		})
	}
}
