package postgres

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"api-aprop/internal/models/event_model"
	"api-aprop/internal/models/preference_model"
	"api-aprop/internal/repositories/postgres/sqlc"
	"api-aprop/internal/repositories/preference_repository"
)

type PreferenceRepository struct {
	pool    *pgxpool.Pool
	queries *sqlc.Queries
}

var _ preference_repository.Repository = (*PreferenceRepository)(nil)

func NewPreferenceRepository(pool *pgxpool.Pool) *PreferenceRepository {
	return &PreferenceRepository{
		pool:    pool,
		queries: sqlc.New(pool),
	}
}

func (r *PreferenceRepository) GetQuestions(ctx context.Context, limit int) ([]preference_model.TestQuestion, error) {
	rows, err := r.queries.GetPreferenceTestQuestions(ctx, int32(limit))
	if err != nil {
		return nil, fmt.Errorf("get preference questions: %w", err)
	}

	questions := make([]preference_model.TestQuestion, 0, len(rows))
	for _, row := range rows {
		var cats []string
		if row.Categories != "" {
			for _, c := range strings.Split(row.Categories, ",") {
				trimmed := strings.TrimSpace(c)
				if trimmed != "" {
					cats = append(cats, trimmed)
				}
			}
		}
		questions = append(questions, preference_model.TestQuestion{
			EventID:     int(row.ID),
			Title:       row.Title,
			Description: row.Description,
			Images:      row.Images,
			Categories:  cats,
		})
	}
	return questions, nil
}

func (r *PreferenceRepository) HasCompletedTest(ctx context.Context, userID string) (bool, error) {
	uid, err := toUUID(userID)
	if err != nil {
		return false, err
	}
	return r.queries.HasCompletedPreferenceTest(ctx, uid)
}

func (r *PreferenceRepository) SaveTestAndPreferences(
	ctx context.Context,
	userID string,
	dateTaken time.Time,
	answers []preference_model.TestAnswer,
	preferences []preference_model.UserPreference,
) error {
	uid, err := toUUID(userID)
	if err != nil {
		return err
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	qtx := r.queries.WithTx(tx)

	// 1. Record test
	ts := pgtype.Timestamp{Time: dateTaken, Valid: true}
	if err := qtx.CreatePreferenceTest(ctx, sqlc.CreatePreferenceTestParams{
		UserID:    uid,
		DateTaken: ts,
	}); err != nil {
		return fmt.Errorf("create preference test: %w", err)
	}

	// 2. Record choices
	for _, a := range answers {
		if err := qtx.CreatePreferenceChoice(ctx, sqlc.CreatePreferenceChoiceParams{
			UserID:    uid,
			DateTaken: ts,
			EventID:   int32(a.EventID),
			Likes:     a.Likes,
		}); err != nil {
			return fmt.Errorf("create preference choice for event %d: %w", a.EventID, err)
		}
	}

	// 3. Upsert preferences
	for _, p := range preferences {
		if err := qtx.UpsertUserPreference(ctx, sqlc.UpsertUserPreferenceParams{
			UserID:     uid,
			CategoryID: int32(p.CategoryID),
			Weight:     p.Weight,
		}); err != nil {
			return fmt.Errorf("upsert preference for category %d: %w", p.CategoryID, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit preference test: %w", err)
	}
	return nil
}

func (r *PreferenceRepository) GetEventCategories(ctx context.Context, eventIDs []int) (map[int][]int, error) {
	ids := make([]int32, len(eventIDs))
	for i, id := range eventIDs {
		ids[i] = int32(id)
	}

	rows, err := r.queries.GetCategoriesForEvents(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("get categories for events: %w", err)
	}

	m := make(map[int][]int)
	for _, row := range rows {
		m[int(row.Event)] = append(m[int(row.Event)], int(row.Category))
	}
	return m, nil
}

func (r *PreferenceRepository) ListAllCategories(ctx context.Context) ([]event_model.Category, error) {
	// Reusing GetCategories from EventRepository or querying directly
	rows, err := r.pool.Query(ctx, "SELECT id, name FROM Categories ORDER BY id ASC")
	if err != nil {
		return nil, fmt.Errorf("list all categories: %w", err)
	}
	defer rows.Close()

	var categories []event_model.Category
	for rows.Next() {
		var c event_model.Category
		if err := rows.Scan(&c.ID, &c.Name); err != nil {
			return nil, err
		}
		categories = append(categories, c)
	}
	return categories, rows.Err()
}

func (r *PreferenceRepository) GetUserPreferences(ctx context.Context, userID string) ([]preference_model.UserPreference, error) {
	uid, err := toUUID(userID)
	if err != nil {
		return nil, err
	}

	rows, err := r.queries.GetUserPreferences(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("get user preferences: %w", err)
	}

	prefs := make([]preference_model.UserPreference, 0, len(rows))
	for _, row := range rows {
		prefs = append(prefs, preference_model.UserPreference{
			CategoryID:   int(row.Category),
			CategoryName: row.CategoryName,
			Weight:       row.Weight,
		})
	}
	return prefs, nil
}
