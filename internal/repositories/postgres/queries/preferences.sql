-- name: GetPreferenceTestQuestions :many
-- Selects events with non-empty title and images, diversified by primary category
-- (round-robin random per category) and returned in random order.
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
ORDER BY random();

-- name: HasCompletedPreferenceTest :one
SELECT EXISTS (
    SELECT 1 FROM PreferenceTests
    WHERE UserID = sqlc.arg(user_id)::uuid
);

-- name: GetPreferenceTestStatus :one
SELECT DateTaken
FROM PreferenceTests
WHERE UserID = sqlc.arg(user_id)::uuid
ORDER BY DateTaken DESC
LIMIT 1;

-- name: LockUserPreferenceTest :exec
-- Serialises concurrent submits of the same user within a transaction so R1 holds.
SELECT pg_advisory_xact_lock(hashtext(sqlc.arg(lock_key)::text));

-- name: DeletePreferenceTestChoicesByUser :exec
DELETE FROM PreferenceTestChoices
WHERE TestUser = sqlc.arg(user_id)::uuid;

-- name: DeletePreferenceTestsByUser :exec
DELETE FROM PreferenceTests
WHERE UserID = sqlc.arg(user_id)::uuid;

-- name: CreatePreferenceTest :exec
INSERT INTO PreferenceTests (UserID, DateTaken)
VALUES (sqlc.arg(user_id)::uuid, sqlc.arg(date_taken)::timestamp);

-- name: CreatePreferenceChoice :exec
INSERT INTO PreferenceTestChoices (TestUser, DateTaken, Event, Likes)
VALUES (sqlc.arg(user_id)::uuid, sqlc.arg(date_taken)::timestamp, sqlc.arg(event_id)::integer, sqlc.arg(likes)::boolean);

-- name: UpsertUserPreference :exec
INSERT INTO UserPreferences (UserID, Category, Weight)
VALUES (sqlc.arg(user_id)::uuid, sqlc.arg(category_id)::integer, sqlc.arg(weight)::real)
ON CONFLICT (UserID, Category)
DO UPDATE SET Weight = EXCLUDED.Weight;

-- name: GetUserPreferences :many
SELECT up.category, c.name AS category_name, up.weight
FROM UserPreferences up
JOIN Categories c ON c.id = up.category
WHERE up.UserID = sqlc.arg(user_id)::uuid
ORDER BY up.weight DESC, c.name ASC;

-- name: GetCategoriesForEvents :many
SELECT ec.event, ec.category, c.name AS category_name
FROM EventCategories ec
JOIN Categories c ON c.id = ec.category
WHERE ec.event = ANY(sqlc.arg(event_ids)::int[]);

-- name: ListAllCategories :many
SELECT id, name
FROM Categories
ORDER BY id ASC;

-- name: GetPreferenceTestQuestionsExcluding :many
-- Selects diversified events with non-empty title and images excluding already-answered event IDs.
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
      AND NOT (e.id = ANY(sqlc.arg(exclude_event_ids)::int[]))
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
    LIMIT sqlc.arg(question_limit)::integer
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
ORDER BY random();

-- name: GetPreferenceTestFullStatus :one
SELECT
    COALESCE((SELECT u.InterestTestPrompted FROM Users u WHERE u.ID = sqlc.arg(user_id)::uuid), false)::boolean AS prompted,
    (SELECT pt.DateTaken FROM PreferenceTests pt WHERE pt.UserID = sqlc.arg(user_id)::uuid ORDER BY pt.DateTaken DESC LIMIT 1) AS date_taken,
    (SELECT COUNT(*)::integer FROM PreferenceTestProgress p WHERE p.UserID = sqlc.arg(user_id)::uuid) AS answered_count;

-- name: MarkUserInterestTestPrompted :exec
UPDATE Users
SET InterestTestPrompted = true
WHERE ID = sqlc.arg(user_id)::uuid;

-- name: ListPreferenceTestProgress :many
SELECT Event, Likes
FROM PreferenceTestProgress
WHERE UserID = sqlc.arg(user_id)::uuid
ORDER BY UpdatedAt ASC, Event ASC;

-- name: UpsertPreferenceTestProgress :exec
INSERT INTO PreferenceTestProgress (UserID, Event, Likes, UpdatedAt)
VALUES (sqlc.arg(user_id)::uuid, sqlc.arg(event_id)::integer, sqlc.arg(likes)::boolean, sqlc.arg(updated_at)::timestamp)
ON CONFLICT (UserID, Event)
DO UPDATE SET Likes = EXCLUDED.Likes, UpdatedAt = EXCLUDED.UpdatedAt;

-- name: DeletePreferenceTestProgressByUser :exec
DELETE FROM PreferenceTestProgress
WHERE UserID = sqlc.arg(user_id)::uuid;
