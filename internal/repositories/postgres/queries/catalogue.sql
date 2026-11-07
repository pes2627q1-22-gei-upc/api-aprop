-- name: SearchEvents :many
-- Events matching the optional filters that overlap the [from_date, to_date] range.
-- The category also matches the title and description; matching ignores accents (unaccent extension).
SELECT e.* FROM Events e
WHERE (sqlc.narg(municipality)::text IS NULL OR unaccent(e.Municipality) ILIKE '%' || unaccent(sqlc.narg(municipality)::text) || '%')
AND (
    sqlc.narg(category)::text IS NULL
    OR EXISTS (
        SELECT 1 FROM EventCategories ec
        JOIN Categories c ON c.ID = ec.Category
        WHERE ec.Event = e.ID AND unaccent(c.Name) ILIKE '%' || unaccent(sqlc.narg(category)::text) || '%'
    )
    OR unaccent(e.Title) ILIKE '%' || unaccent(sqlc.narg(category)::text) || '%'
    OR unaccent(e.Subtitle) ILIKE '%' || unaccent(sqlc.narg(category)::text) || '%'
    OR unaccent(e.Description) ILIKE '%' || unaccent(sqlc.narg(category)::text) || '%'
)
AND (sqlc.narg(to_date)::date IS NULL OR e.StartDate <= sqlc.narg(to_date)::date)
AND (sqlc.narg(from_date)::date IS NULL OR COALESCE(e.EndDate, e.StartDate) >= sqlc.narg(from_date)::date)
-- Relevance of the text (title 3, category 2, subtitle 1, description 1) plus the largest preference weight
-- of the user over the event categories (1 when unknown), then soonest first.
ORDER BY (
    CASE WHEN sqlc.narg(category)::text IS NULL THEN 0 ELSE
        (CASE WHEN unaccent(e.Title) ILIKE '%' || unaccent(sqlc.narg(category)::text) || '%' THEN 3 ELSE 0 END)
        + (CASE WHEN EXISTS (
            SELECT 1 FROM EventCategories ec JOIN Categories c ON c.ID = ec.Category
            WHERE ec.Event = e.ID AND unaccent(c.Name) ILIKE '%' || unaccent(sqlc.narg(category)::text) || '%'
        ) THEN 2 ELSE 0 END)
        + (CASE WHEN unaccent(e.Subtitle) ILIKE '%' || unaccent(sqlc.narg(category)::text) || '%' THEN 1 ELSE 0 END)
        + (CASE WHEN unaccent(e.Description) ILIKE '%' || unaccent(sqlc.narg(category)::text) || '%' THEN 1 ELSE 0 END)
    END
    + COALESCE((
        SELECT MAX(up.Weight) FROM EventCategories ec
        JOIN UserPreferences up ON up.Category = ec.Category
        WHERE ec.Event = e.ID AND up.UserID = sqlc.narg(user_id)::uuid
    ), 1)
) DESC, e.StartDate NULLS LAST, e.ID
LIMIT sqlc.arg(max_results)::integer;

-- name: ListEventsByIDs :many
SELECT * FROM Events
WHERE ID = ANY(sqlc.arg(ids)::integer[]);
