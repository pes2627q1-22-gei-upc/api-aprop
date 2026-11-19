-- name: ListEventTranslations :many
-- Cached translations of the events into lang. SourceHash identifies the Catalan text that was translated.
SELECT Event, Title, Description, SourceHash
FROM EventTranslations
WHERE Lang = sqlc.arg(lang)::varchar AND Event = ANY(sqlc.arg(event_ids)::integer[]);

-- name: UpsertEventTranslation :exec
INSERT INTO EventTranslations (Event, Lang, Title, Description, SourceHash, UpdatedAt)
VALUES (sqlc.arg(event_id)::integer, sqlc.arg(lang)::varchar, sqlc.arg(title)::varchar, sqlc.arg(description)::varchar, sqlc.arg(source_hash)::varchar, now())
ON CONFLICT (Event, Lang) DO UPDATE
SET Title = EXCLUDED.Title, Description = EXCLUDED.Description, SourceHash = EXCLUDED.SourceHash, UpdatedAt = EXCLUDED.UpdatedAt;

-- name: GetUserLanguage :one
-- Language chosen by the user in its settings.
SELECT s.Language
FROM Users u
JOIN Settings s ON s.ID = u.Settings
WHERE u.ID = sqlc.arg(id)::uuid;

-- name: GetSettingsLanguage :one
SELECT Language FROM Settings WHERE ID = sqlc.arg(id)::integer;

-- name: UpdateSettings :exec
-- Settings and language in a single statement (the generated Setting model does not have Language).
UPDATE Settings SET
    ShowPlans = sqlc.arg(show_plans)::boolean,
    NotificationBefore = (sqlc.arg(notification_before)::varchar)::interval,
    Language = sqlc.arg(language)::varchar
WHERE ID = sqlc.arg(id)::integer;
