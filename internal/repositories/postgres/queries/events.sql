-- name: CreateEvent :one
INSERT INTO events (
    title, subtitle, description, schedule, url, tickets, ticketsurl, free, permanent, modality,
    relatedurls, urlnames, images, videos, email, place, latitude, longitude, postalcode, municipality,
    region, address, phone, activityurl, startdate, enddate
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
    $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26
)
RETURNING *;

-- name: UpdateEvent :one
UPDATE events SET
    title = $2, subtitle = $3, description = $4, schedule = $5, url = $6, tickets = $7,
    ticketsurl = $8, free = $9, permanent = $10, modality = $11, relatedurls = $12, urlnames = $13,
    images = $14, videos = $15, email = $16, place = $17, latitude = $18, longitude = $19, postalcode = $20,
    municipality = $21, region = $22, address = $23, phone = $24, activityurl = $25, startdate = $26, enddate = $27
WHERE id = $1
RETURNING *;

-- name: EventExists :one
SELECT EXISTS (SELECT 1 FROM events WHERE id = $1);

-- name: GetEvent :one
SELECT * FROM events
WHERE id = $1;

-- name: ListEvents :many
-- TODO: add paging and filters
SELECT * FROM events
ORDER BY id
LIMIT 10;
