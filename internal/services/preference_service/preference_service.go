package preference_service

import (
	"context"
	"fmt"
	"time"

	"api-aprop/internal/models/preference_model"
	"api-aprop/internal/models/user_model"
	"api-aprop/internal/repositories/preference_repository"
)

const (
	defaultTestLimit = 25
	defaultWeight    = float32(1.0)
	weightStep       = float32(0.2)
	minWeight        = float32(0.1)
	maxWeight        = float32(3.0)
)

type PreferenceService struct {
	repo preference_repository.Repository
	now  func() time.Time
}

var _ Service = (*PreferenceService)(nil)

func NewPreferenceService(repo preference_repository.Repository) *PreferenceService {
	return &PreferenceService{
		repo: repo,
		now:  time.Now,
	}
}

func (s *PreferenceService) GetQuestions(ctx context.Context, userID string) ([]preference_model.TestQuestion, error) {
	if _, err := user_model.NormalizeID(userID); err != nil {
		return nil, err
	}

	questions, err := s.repo.GetQuestions(ctx, defaultTestLimit)
	if err != nil {
		return nil, err
	}
	if questions == nil {
		questions = []preference_model.TestQuestion{}
	}
	return questions, nil
}

func (s *PreferenceService) GetTestStatus(ctx context.Context, userID string) (preference_model.TestStatus, error) {
	normalizedUID, err := user_model.NormalizeID(userID)
	if err != nil {
		return preference_model.TestStatus{}, err
	}
	return s.repo.GetTestStatus(ctx, normalizedUID)
}

func (s *PreferenceService) SubmitTest(ctx context.Context, userID string, answers []preference_model.TestAnswer) (preference_model.SubmitTestResponse, error) {
	normalizedUID, err := user_model.NormalizeID(userID)
	if err != nil {
		return preference_model.SubmitTestResponse{}, err
	}

	if len(answers) == 0 {
		return preference_model.SubmitTestResponse{}, ErrEmptyAnswers
	}
	if len(answers) > defaultTestLimit {
		return preference_model.SubmitTestResponse{}, ErrTooManyAnswers
	}

	eventIDs := make([]int, 0, len(answers))
	for _, a := range answers {
		if a.EventID <= 0 {
			return preference_model.SubmitTestResponse{}, fmt.Errorf("%w: %d", ErrInvalidEventID, a.EventID)
		}
		eventIDs = append(eventIDs, a.EventID)
	}

	// 1. Fetch categories for these events
	eventCategories, err := s.repo.GetEventCategories(ctx, eventIDs)
	if err != nil {
		return preference_model.SubmitTestResponse{}, err
	}

	// 2. Fetch all system categories so every category gets a weight
	allCats, err := s.repo.ListAllCategories(ctx)
	if err != nil {
		return preference_model.SubmitTestResponse{}, err
	}

	// 3. Count likes and dislikes per category
	likesCount := make(map[int]int)
	dislikesCount := make(map[int]int)

	for _, a := range answers {
		cats := eventCategories[a.EventID]
		for _, cat := range cats {
			if a.Likes {
				likesCount[cat]++
			} else {
				dislikesCount[cat]++
			}
		}
	}

	// 4. Calculate weight for each category
	preferences := make([]preference_model.UserPreference, 0, len(allCats))
	for _, c := range allCats {
		likes := likesCount[c.ID]
		dislikes := dislikesCount[c.ID]

		weight := defaultWeight
		if likes+dislikes > 0 {
			weight = defaultWeight + float32(likes-dislikes)*weightStep
			if weight < minWeight {
				weight = minWeight
			}
			if weight > maxWeight {
				weight = maxWeight
			}
		}

		preferences = append(preferences, preference_model.UserPreference{
			CategoryID:   c.ID,
			CategoryName: c.Name,
			Weight:       weight,
		})
	}

	dateTaken := s.now().UTC()
	if err := s.repo.SaveTestAndPreferences(ctx, normalizedUID, dateTaken, answers, preferences); err != nil {
		return preference_model.SubmitTestResponse{}, err
	}

	return preference_model.SubmitTestResponse{
		UserID:      normalizedUID,
		DateTaken:   dateTaken,
		Preferences: preferences,
	}, nil
}

func (s *PreferenceService) GetUserPreferences(ctx context.Context, userID string) ([]preference_model.UserPreference, error) {
	normalizedUID, err := user_model.NormalizeID(userID)
	if err != nil {
		return nil, err
	}

	prefs, err := s.repo.GetUserPreferences(ctx, normalizedUID)
	if err != nil {
		return nil, err
	}
	if prefs == nil {
		prefs = []preference_model.UserPreference{}
	}
	return prefs, nil
}
