// Package translation_repository defines the persistence of the event translations and of the language of the users.
package translation_repository

import "context"

// Translation is the cached translation of the title and description of an event.
// SourceHash identifies the original (Catalan) text that was translated.
type Translation struct {
	Title       string
	Description string
	SourceHash  string
}

type Repository interface {
	// GetTranslations returns the cached translations into lang, by event ID. Events without one are missing.
	GetTranslations(ctx context.Context, lang string, eventIDs []int) (map[int]Translation, error)
	// SaveTranslation stores or replaces the translation of an event into lang.
	SaveTranslation(ctx context.Context, lang string, eventID int, t Translation) error
	// GetUserLanguage returns the language in the settings of the user (userID is its UUID).
	GetUserLanguage(ctx context.Context, userID string) (string, error)
}
