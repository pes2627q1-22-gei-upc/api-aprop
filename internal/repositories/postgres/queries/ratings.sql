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
-- Changes the score of the user's own rating; RatingDate is kept (DP-036).
UPDATE EventRatings SET Rating = sqlc.arg(rating)::integer
WHERE UserID = sqlc.arg(user_id)::uuid AND Event = sqlc.arg(event_id)::integer;

-- name: DeleteEventRating :execrows
DELETE FROM EventRatings
WHERE UserID = sqlc.arg(user_id)::uuid AND Event = sqlc.arg(event_id)::integer;

-- name: CreateUserRating :execrows
INSERT INTO UserRatings (rater, rated, ratingdate, event, rating)
VALUES (sqlc.arg(rater)::uuid, sqlc.arg(rated)::uuid, now(), sqlc.arg(event)::integer, sqlc.arg(rating)::integer)
ON CONFLICT (rater, rated, event) DO NOTHING;

-- name: UpdateUserRating :execrows
UPDATE UserRatings SET Rating=sqlc.arg(rating)::integer
WHERE rater=sqlc.arg(rater)::uuid and rated=sqlc.arg(rated)::uuid;

-- name: DeleteUserRating :execrows
DELETE FROM UserRatings
WHERE rater=sqlc.arg(rater)::uuid and rated=sqlc.arg(rated)::uuid;