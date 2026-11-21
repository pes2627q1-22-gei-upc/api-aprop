package preference_service

import (
	"context"

	"api-aprop/internal/models/preference_model"
)

type Service interface {
	GetQuestions(ctx context.Context, userID string) ([]preference_model.TestQuestion, error)
	GetTestStatus(ctx context.Context, userID string) (preference_model.TestStatus, error)
	SubmitTest(ctx context.Context, userID string, answers []preference_model.TestAnswer) (preference_model.SubmitTestResponse, error)
	GetUserPreferences(ctx context.Context, userID string) ([]preference_model.UserPreference, error)
}
