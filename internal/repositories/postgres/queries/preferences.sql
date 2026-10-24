-- name: GetPreferenceTestQuestions :many
SELECT
    e.id, e.title, e.description, e.images,
    COALESCE(
        (SELECT string_agg(c.name, ',')
         FROM EventCategories ec
         JOIN Categories c ON c.id = ec.category
         WHERE ec.event = e.id),
        ''
    )::text AS categories
FROM Events e
WHERE e.title != ''
ORDER BY e.id
LIMIT $1;

-- name: HasCompletedPreferenceTest :one
SELECT EXISTS (
    SELECT 1 FROM PreferenceTests
    WHERE UserID = sqlc.arg(user_id)::uuid
);

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
