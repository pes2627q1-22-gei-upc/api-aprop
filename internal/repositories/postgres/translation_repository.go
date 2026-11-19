package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"api-aprop/internal/models/user_model"
	"api-aprop/internal/repositories/postgres/sqlc"
	"api-aprop/internal/repositories/translation_repository"
)

// TranslationRepository stores the event translations in Postgres using the sqlc queries.
type TranslationRepository struct {
	queries *sqlc.Queries
}

var _ translation_repository.Repository = (*TranslationRepository)(nil)

func NewTranslationRepository(pool *pgxpool.Pool) *TranslationRepository {
	return &TranslationRepository{queries: sqlc.New(pool)}
}

func (r *TranslationRepository) GetTranslations(ctx context.Context, lang string, eventIDs []int) (map[int]translation_repository.Translation, error) {
	ids := make([]int32, len(eventIDs))
	for i, id := range eventIDs {
		ids[i] = int32(id)
	}
	rows, err := r.queries.ListEventTranslations(ctx, sqlc.ListEventTranslationsParams{Lang: lang, EventIds: ids})
	if err != nil {
		return nil, fmt.Errorf("list event translations: %w", err)
	}
	out := make(map[int]translation_repository.Translation, len(rows))
	for _, row := range rows {
		out[int(row.Event)] = translation_repository.Translation{Title: row.Title, Description: row.Description, SourceHash: row.Sourcehash}
	}
	return out, nil
}

func (r *TranslationRepository) SaveTranslation(ctx context.Context, lang string, eventID int, t translation_repository.Translation) error {
	err := r.queries.UpsertEventTranslation(ctx, sqlc.UpsertEventTranslationParams{
		EventID: int32(eventID), Lang: lang, Title: t.Title, Description: t.Description, SourceHash: t.SourceHash,
	})
	if err != nil {
		return fmt.Errorf("save translation of event %d into %s: %w", eventID, lang, err)
	}
	return nil
}

func (r *TranslationRepository) GetUserLanguage(ctx context.Context, userID string) (string, error) {
	id, err := user_model.ParseID(userID)
	if err != nil {
		return "", err
	}
	lang, err := r.queries.GetUserLanguage(ctx, id)
	if err != nil {
		return "", fmt.Errorf("get language of user %s: %w", userID, err)
	}
	return lang, nil
}
