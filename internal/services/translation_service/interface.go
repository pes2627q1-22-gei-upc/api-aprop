// Package translation_service translates the free-text fields of events into the language of the user.
package translation_service

import (
	"context"

	"api-aprop/internal/models/event_model"
	"api-aprop/internal/models/preference_model"
)

// Service translates events. It never fails: when a text cannot be translated (the translation server is disabled
// or down), the original Catalan text is kept.
type Service interface {
	// UserLanguage returns the language in the settings of the user (userID is its UUID).
	UserLanguage(ctx context.Context, userID string) (string, error)
	TranslateEvent(ctx context.Context, lang string, e event_model.Event) event_model.Event
	TranslateEvents(ctx context.Context, lang string, events []event_model.Event) []event_model.Event
	TranslateMapEvents(ctx context.Context, lang string, events []event_model.MapEvent) []event_model.MapEvent
	TranslateTestQuestions(ctx context.Context, lang string, questions []preference_model.TestQuestion) []preference_model.TestQuestion
}
