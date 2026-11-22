package postgres

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"api-aprop/internal/models/event_model"
	"api-aprop/internal/models/preference_model"
	"api-aprop/internal/models/user_model"
	"api-aprop/internal/repositories/postgres/sqlc"
	"api-aprop/internal/repositories/preference_repository"
)

// TODO(DP-047): These queries are declared in internal/repositories/postgres/queries/preferences.sql
// and executed directly via pgx until the team runs `sqlc generate` (internal/repositories/postgres/sqlc/*
// must not be modified directly).
const (
	getPreferenceTestQuestionsExcludingSQL = `
WITH candidates AS (
    SELECT
        e.id,
        e.title,
        e.description,
        e.images,
        COALESCE(MIN(ec.category), 0) AS primary_category
    FROM Events e
    LEFT JOIN EventCategories ec ON ec.event = e.id
    WHERE btrim(e.title) != ''
      AND e.images IS NOT NULL
      AND btrim(e.images) != ''
      AND NOT (e.id = ANY($1::int[]))
    GROUP BY e.id, e.title, e.description, e.images
),
ranked AS (
    SELECT
        id,
        title,
        description,
        images,
        primary_category,
        ROW_NUMBER() OVER (
            PARTITION BY primary_category
            ORDER BY random()
        ) AS rn
    FROM candidates
),
selected AS (
    SELECT id, title, description, images
    FROM ranked
    ORDER BY (primary_category = 0) ASC, rn ASC, random()
    LIMIT $2::integer
)
SELECT
    s.id,
    s.title,
    s.description,
    s.images,
    COALESCE(
        (SELECT string_agg(c.name, ',' ORDER BY c.id)
         FROM EventCategories ec
         JOIN Categories c ON c.id = ec.category
         WHERE ec.event = s.id),
        ''
    )::text AS categories
FROM selected s
ORDER BY random()`

	getPreferenceTestFullStatusSQL = `
SELECT
    COALESCE((SELECT u.InterestTestPrompted FROM Users u WHERE u.ID = $1::uuid), false)::boolean AS prompted,
    (SELECT pt.DateTaken FROM PreferenceTests pt WHERE pt.UserID = $1::uuid ORDER BY pt.DateTaken DESC LIMIT 1) AS date_taken,
    (SELECT COUNT(*)::integer FROM PreferenceTestProgress p WHERE p.UserID = $1::uuid) AS answered_count`

	markUserInterestTestPromptedSQL = `
UPDATE Users
SET InterestTestPrompted = true
WHERE ID = $1::uuid`

	listPreferenceTestProgressSQL = `
SELECT Event, Likes
FROM PreferenceTestProgress
WHERE UserID = $1::uuid
ORDER BY UpdatedAt ASC, Event ASC`

	upsertPreferenceTestProgressSQL = `
INSERT INTO PreferenceTestProgress (UserID, Event, Likes, UpdatedAt)
VALUES ($1::uuid, $2::integer, $3::boolean, $4::timestamp)
ON CONFLICT (UserID, Event)
DO UPDATE SET Likes = EXCLUDED.Likes, UpdatedAt = EXCLUDED.UpdatedAt`

	deletePreferenceTestProgressByUserSQL = `
DELETE FROM PreferenceTestProgress
WHERE UserID = $1::uuid`
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

	rows, err := r.pool.Query(ctx, getPreferenceTestQuestionsExcludingSQL, toInt32Slice(excludeEventIDs), int32(limit))
	if err != nil {
		return nil, fmt.Errorf("get preference questions excluding answered: %w", err)
	}
	defer rows.Close()

	questions := make([]preference_model.TestQuestion, 0, limit)
	for rows.Next() {
		var (
			id          int32
			title       string
			description pgtype.Text
			images      pgtype.Text
			categories  string
		)
		if err := rows.Scan(&id, &title, &description, &images, &categories); err != nil {
			return nil, fmt.Errorf("scan preference question: %w", err)
		}
		questions = append(questions, newTestQuestion(id, title, description, images, categories))
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate preference questions: %w", err)
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

	var (
		prompted      bool
		ts            pgtype.Timestamp
		answeredCount int32
	)
	if err := r.pool.QueryRow(ctx, getPreferenceTestFullStatusSQL, uid).Scan(&prompted, &ts, &answeredCount); err != nil {
		return preference_model.TestStatus{}, fmt.Errorf("get preference test status: %w", err)
	}

	var dateTaken *time.Time
	completed := false
	if ts.Valid {
		t := ts.Time.UTC()
		dateTaken = &t
		completed = true
	}
	inProgress := answeredCount > 0
	return preference_model.TestStatus{
		Completed:     completed,
		Prompted:      prompted || completed || inProgress,
		InProgress:    inProgress,
		AnsweredCount: int(answeredCount),
		DateTaken:     dateTaken,
	}, nil
}

func (r *PreferenceRepository) MarkTestPrompted(ctx context.Context, userID string) error {
	uid, err := user_model.ParseID(userID)
	if err != nil {
		return err
	}
	if _, err := r.pool.Exec(ctx, markUserInterestTestPromptedSQL, uid); err != nil {
		return fmt.Errorf("mark preference test prompted: %w", err)
	}
	return nil
}

func (r *PreferenceRepository) GetTestProgress(ctx context.Context, userID string) ([]preference_model.TestAnswer, error) {
	uid, err := user_model.ParseID(userID)
	if err != nil {
		return nil, err
	}

	rows, err := r.pool.Query(ctx, listPreferenceTestProgressSQL, uid)
	if err != nil {
		return nil, fmt.Errorf("list preference test progress: %w", err)
	}
	defer rows.Close()

	answers := make([]preference_model.TestAnswer, 0)
	for rows.Next() {
		var (
			eventID int32
			likes   bool
		)
		if err := rows.Scan(&eventID, &likes); err != nil {
			return nil, fmt.Errorf("scan preference test progress: %w", err)
		}
		answers = append(answers, preference_model.TestAnswer{
			EventID: int(eventID),
			Likes:   likes,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate preference test progress: %w", err)
	}
	return answers, nil
}

func (r *PreferenceRepository) withLockedUserTx(
	ctx context.Context,
	userID string,
	commitOp string,
	fn func(tx pgx.Tx, qtx *sqlc.Queries, uid pgtype.UUID) error,
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

	if err := fn(tx, qtx, uid); err != nil {
		return err
	}

	if _, err := tx.Exec(ctx, markUserInterestTestPromptedSQL, uid); err != nil {
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
	return r.withLockedUserTx(ctx, userID, "commit partial preference progress", func(tx pgx.Tx, _ *sqlc.Queries, uid pgtype.UUID) error {
		ts := pgtype.Timestamp{Time: updatedAt.UTC(), Valid: true}
		for _, a := range answers {
			if _, err := tx.Exec(ctx, upsertPreferenceTestProgressSQL, uid, int32(a.EventID), a.Likes, ts); err != nil {
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
	return r.withLockedUserTx(ctx, userID, "commit preference test", func(tx pgx.Tx, qtx *sqlc.Queries, uid pgtype.UUID) error {
		// 0. Replace previous test, choices and any draft progress so each user keeps a single active TestInteressos (R1).
		// TODO(DP-044): R1 s'aplica per esborrat + inserció dins la transacció (sense UNIQUE(UserID) a la BD).
		if err := qtx.DeletePreferenceTestChoicesByUser(ctx, uid); err != nil {
			return fmt.Errorf("delete previous preference choices: %w", err)
		}
		if err := qtx.DeletePreferenceTestsByUser(ctx, uid); err != nil {
			return fmt.Errorf("delete previous preference tests: %w", err)
		}
		if _, err := tx.Exec(ctx, deletePreferenceTestProgressByUserSQL, uid); err != nil {
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
