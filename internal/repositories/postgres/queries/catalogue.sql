-- name: SearchEvents :many
-- Events matching the optional filters that overlap the [from_date, to_date] range.
-- Categories are not imported from the open data yet, so the category also matches the title and description.
-- TODO(DP-018): matching is accent sensitive (needs the unaccent extension or a normalized column).
SELECT e.* FROM Events e
WHERE (sqlc.narg(municipality)::text IS NULL OR e.Municipality ILIKE '%' || sqlc.narg(municipality)::text || '%')
AND (
    sqlc.narg(category)::text IS NULL
    OR EXISTS (
        SELECT 1 FROM EventCategories ec
        JOIN Categories c ON c.ID = ec.Category
        WHERE ec.Event = e.ID AND c.Name ILIKE '%' || sqlc.narg(category)::text || '%'
    )
    OR e.Title ILIKE '%' || sqlc.narg(category)::text || '%'
    OR e.Subtitle ILIKE '%' || sqlc.narg(category)::text || '%'
    OR e.Description ILIKE '%' || sqlc.narg(category)::text || '%'
)
AND (sqlc.narg(to_date)::date IS NULL OR e.StartDate <= sqlc.narg(to_date)::date)
AND (sqlc.narg(from_date)::date IS NULL OR COALESCE(e.EndDate, e.StartDate) >= sqlc.narg(from_date)::date)
ORDER BY e.StartDate NULLS LAST, e.ID
LIMIT sqlc.arg(max_results)::integer;

-- name: ListEventsByIDs :many
SELECT * FROM Events
WHERE ID = ANY(sqlc.arg(ids)::integer[]);
