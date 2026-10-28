package preference_repository

import (
	"context"
	"time"

	"api-aprop/internal/models/event_model"
	"api-aprop/internal/models/preference_model"
)

type Repository interface {
	GetQuestions(ctx context.Context, limit int) ([]preference_model.TestQuestion, error)
	HasCompletedTest(ctx context.Context, userID string) (bool, error)
	SaveTestAndPreferences(ctx context.Context, userID string, dateTaken time.Time, answers []preference_model.TestAnswer, preferences []preference_model.UserPreference) error
	GetEventCategories(ctx context.Context, eventIDs []int) (map[int][]int, error)
	ListAllCategories(ctx context.Context) ([]event_model.Category, error)
	GetUserPreferences(ctx context.Context, userID string) ([]preference_model.UserPreference, error)
}
