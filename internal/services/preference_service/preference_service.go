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
	normalizedUID, err := user_model.NormalizeID(userID)
	if err != nil {
		return nil, err
	}

	progress, err := s.repo.GetTestProgress(ctx, normalizedUID)
	if err != nil {
		return nil, err
	}

	var questions []preference_model.TestQuestion
	if len(progress) == 0 {
		questions, err = s.repo.GetQuestions(ctx, defaultTestLimit)
	} else {
		remaining := defaultTestLimit - len(progress)
		if remaining <= 0 {
			return []preference_model.TestQuestion{}, nil
		}
		excludeIDs := make([]int, len(progress))
		for i, a := range progress {
			excludeIDs[i] = a.EventID
		}
		questions, err = s.repo.GetQuestionsExcluding(ctx, remaining, excludeIDs)
	}
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

func (s *PreferenceService) MarkTestPrompted(ctx context.Context, userID string) error {
	normalizedUID, err := user_model.NormalizeID(userID)
	if err != nil {
		return err
	}
	return s.repo.MarkTestPrompted(ctx, normalizedUID)
}

type preparedAnswers struct {
	userID  string
	deduped []preference_model.TestAnswer
	merged  []preference_model.TestAnswer
}

func (s *PreferenceService) withPreparedAnswers(
	ctx context.Context,
	userID string,
	answers []preference_model.TestAnswer,
	fn func(preparedAnswers) (preference_model.SubmitTestResponse, error),
) (preference_model.SubmitTestResponse, error) {
	normalizedUID, err := user_model.NormalizeID(userID)
	if err != nil {
		return preference_model.SubmitTestResponse{}, err
	}

	defer lockUser(normalizedUID)()

	deduped, err := validateAndDeduplicateAnswers(answers)
	if err != nil {
		return preference_model.SubmitTestResponse{}, err
	}

	existing, err := s.repo.GetTestProgress(ctx, normalizedUID)
	if err != nil {
		return preference_model.SubmitTestResponse{}, err
	}

	merged := mergeAnswers(existing, deduped)
	if len(merged) > defaultTestLimit {
		return preference_model.SubmitTestResponse{}, ErrTooManyAnswers
	}

	return fn(preparedAnswers{
		userID:  normalizedUID,
		deduped: deduped,
		merged:  merged,
	})
}

func (s *PreferenceService) SavePartialTest(ctx context.Context, userID string, answers []preference_model.TestAnswer) (preference_model.SubmitTestResponse, error) {
	return s.withPreparedAnswers(ctx, userID, answers, func(p preparedAnswers) (preference_model.SubmitTestResponse, error) {
		updatedAt := s.now().UTC()
		// TODO(DP-048): Partial progress is persisted in PreferenceTestProgress without touching
		// PreferenceTests / UserPreferences so prior completed answers stay intact during a paused retake.
		if err := s.repo.SavePartialProgress(ctx, p.userID, updatedAt, p.deduped); err != nil {
			return preference_model.SubmitTestResponse{}, err
		}

		return preference_model.SubmitTestResponse{
			UserID:        p.userID,
			DateTaken:     updatedAt,
			Preferences:   []preference_model.UserPreference{},
			Partial:       true,
			AnsweredCount: len(p.merged),
		}, nil
	})
}

func (s *PreferenceService) SubmitTest(ctx context.Context, userID string, answers []preference_model.TestAnswer) (preference_model.SubmitTestResponse, error) {
	return s.withPreparedAnswers(ctx, userID, answers, func(p preparedAnswers) (preference_model.SubmitTestResponse, error) {
		eventIDs := make([]int, 0, len(p.merged))
		for _, a := range p.merged {
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

		// 3. Count likes and dislikes per category across all accumulated answers
		likesCount := make(map[int]int)
		dislikesCount := make(map[int]int)

		for _, a := range p.merged {
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
		if err := s.repo.SaveTestAndPreferences(ctx, p.userID, dateTaken, p.merged, preferences); err != nil {
			return preference_model.SubmitTestResponse{}, err
		}

		return preference_model.SubmitTestResponse{
			UserID:      p.userID,
			DateTaken:   dateTaken,
			Preferences: preferences,
		}, nil
	})
}

func validateAndDeduplicateAnswers(answers []preference_model.TestAnswer) ([]preference_model.TestAnswer, error) {
	if len(answers) == 0 {
		return nil, ErrEmptyAnswers
	}
	if len(answers) > defaultTestLimit {
		return nil, ErrTooManyAnswers
	}
	for _, a := range answers {
		if a.EventID <= 0 {
			return nil, fmt.Errorf("%w: %d", ErrInvalidEventID, a.EventID)
		}
	}
	return mergeAnswers(nil, answers), nil
}

func mergeAnswers(existing, incoming []preference_model.TestAnswer) []preference_model.TestAnswer {
	merged := make([]preference_model.TestAnswer, 0, len(existing)+len(incoming))
	indexByEvent := make(map[int]int, len(existing)+len(incoming))
	appendUnique := func(batch []preference_model.TestAnswer) {
		for _, a := range batch {
			if idx, exists := indexByEvent[a.EventID]; exists {
				merged[idx].Likes = a.Likes
				continue
			}
			indexByEvent[a.EventID] = len(merged)
			merged = append(merged, a)
		}
	}
	appendUnique(existing)
	appendUnique(incoming)
	return merged
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
