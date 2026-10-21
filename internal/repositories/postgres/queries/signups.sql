-- name: CreateSignUp :execrows
-- Inserts the sign up only if the user is not already signed up to the event.
-- TODO(DP-012): without a UNIQUE (UserID, Event) constraint, concurrent requests can still insert duplicates.
INSERT INTO SignUp (UserID, Event, SignUpDate)
SELECT sqlc.arg(user_id)::uuid, sqlc.arg(event_id)::integer, now()
WHERE NOT EXISTS (
    SELECT 1 FROM SignUp
    WHERE UserID = sqlc.arg(user_id)::uuid AND Event = sqlc.arg(event_id)::integer
);

-- name: ListUserSignedUpEvents :many
-- Events the user is signed up to that overlap the optional [from_date, to_date] range.
-- EXISTS returns each event once even if SignUp contains duplicated rows.
-- Modality is an enum not present in the hand-edited models, so it is read as text.
SELECT
    e.id, e.title, e.subtitle, e.description, e.schedule, e.url, e.tickets, e.ticketsurl,
    e.free, e.permanent, COALESCE(e.modality::text, '')::text AS modality, e.relatedurls, e.urlnames, e.images,
    e.videos, e.email, e.place, e.latitude, e.longitude, e.postalcode, e.municipality,
    e.region, e.address, e.phone, e.activityurl, e.startdate, e.enddate
FROM Events e
WHERE EXISTS (
    SELECT 1 FROM SignUp s
    WHERE s.Event = e.ID AND s.UserID = sqlc.arg(user_id)::uuid
)
AND (sqlc.narg(to_date)::date IS NULL OR e.StartDate <= sqlc.narg(to_date)::date)
AND (sqlc.narg(from_date)::date IS NULL OR COALESCE(e.EndDate, e.StartDate) >= sqlc.narg(from_date)::date)
ORDER BY e.StartDate, e.ID;
