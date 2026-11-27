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
	"api-aprop/internal/models/user_model"
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

func parseCategories(raw string) []string {
	cats := make([]string, 0)
	if raw == "" {
		return cats
	}
	for _, c := range strings.Split(raw, ",") {
		trimmed := strings.TrimSpace(c)
		if trimmed != "" {
			cats = append(cats, trimmed)
		}
	}
	return cats
}

func toInt32Slice(ids []int) []int32 {
	out := make([]int32, len(ids))
	for i, id := range ids {
		out[i] = int32(id)
	}
	return out
}

func newTestQuestion(id int32, title string, description, images pgtype.Text, categories string) preference_model.TestQuestion {
	return preference_model.TestQuestion{
		EventID:     int(id),
		Title:       title,
		Description: description.String,
		Images:      images.String,
		Categories:  parseCategories(categories),
	}
}

func (r *PreferenceRepository) GetQuestions(ctx context.Context, limit int) ([]preference_model.TestQuestion, error) {
	if limit <= 0 {
		return []preference_model.TestQuestion{}, nil
	}

	rows, err := r.queries.GetPreferenceTestQuestions(ctx, int32(limit))
	if err != nil {
		return nil, fmt.Errorf("get preference questions: %w", err)
	}

	questions := make([]preference_model.TestQuestion, 0, len(rows))
	for _, row := range rows {
		questions = append(questions, newTestQuestion(row.ID, row.Title, row.Description, row.Images, row.Categories))
	}
	return questions, nil
}

func (r *PreferenceRepository) GetQuestionsExcluding(ctx context.Context, limit int, excludeEventIDs []int) ([]preference_model.TestQuestion, error) {
	if limit <= 0 {
		return []preference_model.TestQuestion{}, nil
	}
	if len(excludeEventIDs) == 0 {
		return r.GetQuestions(ctx, limit)
	}

	rows, err := r.queries.GetPreferenceTestQuestionsExcluding(ctx, sqlc.GetPreferenceTestQuestionsExcludingParams{
		ExcludeEventIds: toInt32Slice(excludeEventIDs),
		QuestionLimit:   int32(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("get preference questions excluding answered: %w", err)
	}

	questions := make([]preference_model.TestQuestion, 0, len(rows))
	for _, row := range rows {
		questions = append(questions, newTestQuestion(row.ID, row.Title, row.Description, row.Images, row.Categories))
	}
	return questions, nil
}

func (r *PreferenceRepository) HasCompletedTest(ctx context.Context, userID string) (bool, error) {
	uid, err := user_model.ParseID(userID)
	if err != nil {
		return false, err
	}
	return r.queries.HasCompletedPreferenceTest(ctx, uid)
}

func (r *PreferenceRepository) GetTestStatus(ctx context.Context, userID string) (preference_model.TestStatus, error) {
	uid, err := user_model.ParseID(userID)
	if err != nil {
		return preference_model.TestStatus{}, err
	}

	row, err := r.queries.GetPreferenceTestFullStatus(ctx, uid)
	if err != nil {
		return preference_model.TestStatus{}, fmt.Errorf("get preference test status: %w", err)
	}

	var dateTaken *time.Time
	completed := false
	if row.DateTaken.Valid {
		t := row.DateTaken.Time.UTC()
		dateTaken = &t
		completed = true
	}
	inProgress := row.AnsweredCount > 0
	return preference_model.TestStatus{
		Completed:     completed,
		Prompted:      row.Prompted || completed || inProgress,
		InProgress:    inProgress,
		AnsweredCount: int(row.AnsweredCount),
		DateTaken:     dateTaken,
	}, nil
}

func (r *PreferenceRepository) MarkTestPrompted(ctx context.Context, userID string) error {
	uid, err := user_model.ParseID(userID)
	if err != nil {
		return err
	}
	if err := r.queries.MarkUserInterestTestPrompted(ctx, uid); err != nil {
		return fmt.Errorf("mark preference test prompted: %w", err)
	}
	return nil
}

func (r *PreferenceRepository) GetTestProgress(ctx context.Context, userID string) ([]preference_model.TestAnswer, error) {
	uid, err := user_model.ParseID(userID)
	if err != nil {
		return nil, err
	}

	rows, err := r.queries.ListPreferenceTestProgress(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("list preference test progress: %w", err)
	}

	answers := make([]preference_model.TestAnswer, 0, len(rows))
	for _, row := range rows {
		answers = append(answers, preference_model.TestAnswer{
			EventID: int(row.Event),
			Likes:   row.Likes,
		})
	}
	return answers, nil
}

func (r *PreferenceRepository) withLockedUserTx(
	ctx context.Context,
	userID string,
	commitOp string,
	fn func(qtx *sqlc.Queries, uid pgtype.UUID) error,
) error {
	uid, err := user_model.ParseID(userID)
	if err != nil {
		return err
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	qtx := r.queries.WithTx(tx)
	// Serialise concurrent submits of the same user so R1 holds (a first test has no row to lock).
	if err := qtx.LockUserPreferenceTest(ctx, uid.String()); err != nil {
		return fmt.Errorf("lock preference test: %w", err)
	}

	if err := fn(qtx, uid); err != nil {
		return err
	}

	if err := qtx.MarkUserInterestTestPrompted(ctx, uid); err != nil {
		return fmt.Errorf("mark preference test prompted: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("%s: %w", commitOp, err)
	}
	return nil
}

func (r *PreferenceRepository) SavePartialProgress(
	ctx context.Context,
	userID string,
	updatedAt time.Time,
	answers []preference_model.TestAnswer,
) error {
	return r.withLockedUserTx(ctx, userID, "commit partial preference progress", func(qtx *sqlc.Queries, uid pgtype.UUID) error {
		ts := pgtype.Timestamp{Time: updatedAt.UTC(), Valid: true}
		for _, a := range answers {
			if err := qtx.UpsertPreferenceTestProgress(ctx, sqlc.UpsertPreferenceTestProgressParams{
				UserID:    uid,
				EventID:   int32(a.EventID),
				Likes:     a.Likes,
				UpdatedAt: ts,
			}); err != nil {
				return fmt.Errorf("upsert partial preference choice for event %d: %w", a.EventID, err)
			}
		}
		return nil
	})
}

func (r *PreferenceRepository) SaveTestAndPreferences(
	ctx context.Context,
	userID string,
	dateTaken time.Time,
	answers []preference_model.TestAnswer,
	preferences []preference_model.UserPreference,
) error {
	return r.withLockedUserTx(ctx, userID, "commit preference test", func(qtx *sqlc.Queries, uid pgtype.UUID) error {
		// 0. Replace previous test, choices and any draft progress so each user keeps a single active TestInteressos (R1).
		// TODO(DP-044): R1 s'aplica per esborrat + inserció dins la transacció (sense UNIQUE(UserID) a la BD).
		if err := qtx.DeletePreferenceTestChoicesByUser(ctx, uid); err != nil {
			return fmt.Errorf("delete previous preference choices: %w", err)
		}
		if err := qtx.DeletePreferenceTestsByUser(ctx, uid); err != nil {
			return fmt.Errorf("delete previous preference tests: %w", err)
		}
		if err := qtx.DeletePreferenceTestProgressByUser(ctx, uid); err != nil {
			return fmt.Errorf("delete partial preference progress: %w", err)
		}

		// 1. Record test
		ts := pgtype.Timestamp{Time: dateTaken.UTC(), Valid: true}
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
		return nil
	})
}

func (r *PreferenceRepository) GetEventCategories(ctx context.Context, eventIDs []int) (map[int][]int, error) {
	rows, err := r.queries.GetCategoriesForEvents(ctx, toInt32Slice(eventIDs))
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
	rows, err := r.queries.ListAllCategories(ctx)
	if err != nil {
		return nil, fmt.Errorf("list all categories: %w", err)
	}

	categories := make([]event_model.Category, 0, len(rows))
	for _, row := range rows {
		categories = append(categories, event_model.Category{
			ID:   int(row.ID),
			Name: row.Name,
		})
	}
	return categories, nil
}

func (r *PreferenceRepository) GetUserPreferences(ctx context.Context, userID string) ([]preference_model.UserPreference, error) {
	uid, err := user_model.ParseID(userID)
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
