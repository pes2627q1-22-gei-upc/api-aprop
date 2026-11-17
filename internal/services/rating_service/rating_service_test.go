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
	created     bool
	createErr   error
	createCalls int
	gotScore    int
}

func (f *fakeRatings) Create(_ context.Context, _ string, _ int, score int) (bool, error) {
	f.createCalls++
	f.gotScore = score
	return f.created, f.createErr
}

func (f *fakeRatings) HasAttended(context.Context, string, int) (bool, error) {
	return f.attended, nil
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
