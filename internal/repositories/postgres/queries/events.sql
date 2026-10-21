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
LIMIT $1 OFFSET $2;

-- name: CreateOfficialEvent :one
INSERT INTO OfficialEvents(id, code) VALUES ($1, $2) RETURNING ID;

-- name: EqualEventExists :one
SELECT id FROM events WHERE title IS NOT DISTINCT FROM $1 
and subtitle IS NOT DISTINCT FROM $2
and description IS NOT DISTINCT FROM $3
and schedule IS NOT DISTINCT FROM $4
and url IS NOT DISTINCT FROM $5
and tickets IS NOT DISTINCT FROM $6
and ticketsurl IS NOT DISTINCT FROM $7
and free IS NOT DISTINCT FROM $8
and permanent IS NOT DISTINCT FROM $9
and modality IS NOT DISTINCT FROM $10
and relatedurls IS NOT DISTINCT FROM $11
and urlnames IS NOT DISTINCT FROM $12
and images IS NOT DISTINCT FROM $13
and videos IS NOT DISTINCT FROM $14
and email IS NOT DISTINCT FROM $15
and place IS NOT DISTINCT FROM $16
and latitude IS NOT DISTINCT FROM $17
and longitude IS NOT DISTINCT FROM $18
and postalcode IS NOT DISTINCT FROM $19
and municipality IS NOT DISTINCT FROM $20
and region IS NOT DISTINCT FROM $21
and address IS NOT DISTINCT FROM $22
and phone IS NOT DISTINCT FROM $23
and activityurl IS NOT DISTINCT FROM $24
and startdate IS NOT DISTINCT FROM $25
and enddate IS NOT DISTINCT FROM $26 LIMIT 1;