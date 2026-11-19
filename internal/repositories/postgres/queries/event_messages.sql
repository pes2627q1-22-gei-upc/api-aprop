-- name: CreateEventMessage :one
-- Stores a chat message of the event and returns it with its author. DateSent is part of the R1 key.
-- message is the text encoded by the repository (format byte + plain or compressed text, DP-040).
WITH m AS (
    INSERT INTO EventMessages (UserID, Event, DateSent, Message)
    VALUES (sqlc.arg(user_id)::uuid, sqlc.arg(event_id)::integer, now(), sqlc.arg(message)::bytea)
    RETURNING UserID, Message, DateSent
)
SELECT m.UserID, u.Username, u.Name, u.ProfilePicture, m.Message, m.DateSent
FROM m
JOIN Users u ON u.ID = m.UserID;

-- name: ListEventMessages :many
-- Chat messages of the event with their authors, newest first.
SELECT m.UserID, u.Username, u.Name, u.ProfilePicture, m.Message, m.DateSent
FROM EventMessages m
JOIN Users u ON u.ID = m.UserID
WHERE m.Event = sqlc.arg(event_id)::integer
ORDER BY m.DateSent DESC, m.UserID
LIMIT sqlc.arg(page_size)::int OFFSET sqlc.arg(page_offset)::int;

-- name: CountEventMessages :one
SELECT COUNT(*)
FROM EventMessages
WHERE Event = sqlc.arg(event_id)::integer;
