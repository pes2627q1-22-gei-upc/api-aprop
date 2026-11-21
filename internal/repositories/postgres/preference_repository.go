package postgres

import (
	"context"
	"errors"
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

// TODO(DP-044): inline SQL per seleccionar preguntes amb títol i imatges no buits i diversitat
// per categoria (round-robin per primera categoria + ordre aleatori), pendent de validar contra Supabase.
const preferenceQuestionsQuery = `
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
	LIMIT $1
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
ORDER BY random()
`

// TODO(DP-044): inline SQL per consultar l'estat i la data del test actiu sense regenerar sqlc.
const preferenceTestStatusQuery = `
SELECT DateTaken
FROM PreferenceTests
WHERE UserID = $1::uuid
ORDER BY DateTaken DESC
LIMIT 1
`

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

func (r *PreferenceRepository) GetQuestions(ctx context.Context, limit int) ([]preference_model.TestQuestion, error) {
	if limit <= 0 {
		return []preference_model.TestQuestion{}, nil
	}

	rows, err := r.pool.Query(ctx, preferenceQuestionsQuery, limit)
	if err != nil {
		return nil, fmt.Errorf("get preference questions: %w", err)
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
		questions = append(questions, preference_model.TestQuestion{
			EventID:     int(id),
			Title:       title,
			Description: description.String,
			Images:      images.String,
			Categories:  parseCategories(categories),
		})
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

	var ts pgtype.Timestamp
	err = r.pool.QueryRow(ctx, preferenceTestStatusQuery, uid).Scan(&ts)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return preference_model.TestStatus{Completed: false, DateTaken: nil}, nil
		}
		return preference_model.TestStatus{}, fmt.Errorf("get preference test status: %w", err)
	}

	var dateTaken *time.Time
	if ts.Valid {
		t := ts.Time.UTC()
		dateTaken = &t
	}
	return preference_model.TestStatus{
		Completed: true,
		DateTaken: dateTaken,
	}, nil
}

func (r *PreferenceRepository) SaveTestAndPreferences(
	ctx context.Context,
	userID string,
	dateTaken time.Time,
	answers []preference_model.TestAnswer,
	preferences []preference_model.UserPreference,
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

	// Serialise concurrent submits of the same user so R1 holds (a first test has no row to lock).
	if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock(hashtext($1::text))", uid.String()); err != nil {
		return fmt.Errorf("lock preference test: %w", err)
	}

	// 0. Replace previous test and choices so each user keeps a single active TestInteressos (R1).
	// TODO(DP-044): R1 s'aplica per esborrat + inserció dins la transacció (sense UNIQUE(UserID) a la BD).
	if _, err := tx.Exec(ctx, "DELETE FROM PreferenceTestChoices WHERE TestUser = $1::uuid", uid); err != nil {
		return fmt.Errorf("delete previous preference choices: %w", err)
	}
	if _, err := tx.Exec(ctx, "DELETE FROM PreferenceTests WHERE UserID = $1::uuid", uid); err != nil {
		return fmt.Errorf("delete previous preference tests: %w", err)
	}

	qtx := r.queries.WithTx(tx)

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
