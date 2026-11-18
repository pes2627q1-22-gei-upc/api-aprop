-- name: CreateEventRating :execrows
-- Inserts the rating unless the user has already rated the event.
INSERT INTO EventRatings (UserID, Event, RatingDate, Rating)
VALUES (sqlc.arg(user_id)::uuid, sqlc.arg(event_id)::integer, now(), sqlc.arg(rating)::integer)
ON CONFLICT (UserID, Event) DO NOTHING;

-- name: HasUserAttendedEvent :one
SELECT EXISTS (
    SELECT 1 FROM SignUp
    WHERE UserID = sqlc.arg(user_id)::uuid AND Event = sqlc.arg(event_id)::integer AND Attendance IS TRUE
)::boolean;

-- name: UpdateEventRating :execrows
-- Changes the score of the user's own rating; RatingDate is kept (DP-034).
UPDATE EventRatings SET Rating = sqlc.arg(rating)::integer
WHERE UserID = sqlc.arg(user_id)::uuid AND Event = sqlc.arg(event_id)::integer;

-- name: DeleteEventRating :execrows
DELETE FROM EventRatings
WHERE UserID = sqlc.arg(user_id)::uuid AND Event = sqlc.arg(event_id)::integer;
