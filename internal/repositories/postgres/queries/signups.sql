-- name: CreateSignUp :execrows
-- Inserts a sign up row if the (UserID, Event) pair does not already exist.
INSERT INTO SignUp (UserID, Event)
SELECT sqlc.arg(user_id)::uuid, sqlc.arg(event_id)::integer
WHERE NOT EXISTS (
    SELECT 1 FROM SignUp
    WHERE UserID = sqlc.arg(user_id)::uuid AND Event = sqlc.arg(event_id)::integer
);

-- name: DeleteSignUp :execrows
-- Deletes the user's sign up for the event.
DELETE FROM SignUp
WHERE UserID = sqlc.arg(user_id)::uuid AND Event = sqlc.arg(event_id)::integer;

-- name: CountEventSignUps :one
SELECT COUNT(DISTINCT UserID)::bigint
FROM SignUp
WHERE Event = sqlc.arg(event_id)::integer;

-- name: IsUserSignedUp :one
SELECT EXISTS (
    SELECT 1 FROM SignUp
    WHERE UserID = sqlc.arg(user_id)::uuid AND Event = sqlc.arg(event_id)::integer
)::boolean;

-- name: ListUserSignedUpEvents :many
-- Events the user is signed up to that overlap the optional [from_date, to_date] range.
-- EXISTS returns each event once even if SignUp contains duplicated rows.
SELECT
    e.id, e.title, e.subtitle, e.description, e.schedule, e.url, e.tickets, e.ticketsurl,
    e.free, e.permanent, e.modality, e.images, e.videos, e.email, e.place, e.latitude,
    e.longitude, e.postalcode, e.municipality, e.region, e.address, e.phone, e.activityurl,
    e.startdate, e.enddate, e.relatedurls, e.urlnames
FROM Events e
WHERE EXISTS (
    SELECT 1 FROM SignUp s
    WHERE s.Event = e.id AND s.UserID = sqlc.arg(user_id)::uuid
)
AND (sqlc.narg(from_date)::timestamptz IS NULL OR e.EndDate >= sqlc.narg(from_date))
AND (sqlc.narg(to_date)::timestamptz IS NULL OR e.StartDate <= sqlc.narg(to_date))
ORDER BY e.StartDate ASC, e.id ASC;

-- name: SetSignUpAttendance :execrows
-- Sets the attendance (NULL = unknown) of the user's sign up for the event.
UPDATE SignUp
SET Attendance = sqlc.narg(attendance)::boolean
WHERE UserID = sqlc.arg(user_id)::uuid AND Event = sqlc.arg(event_id)::integer;
