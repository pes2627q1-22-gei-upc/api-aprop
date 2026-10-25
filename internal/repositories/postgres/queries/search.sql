-- name: SearchEventsWithRelevance :many
SELECT evlst.*, c.name as category FROM (
    SELECT e.id, e.title, e.subtitle, e.description, e.schedule, e.tickets, e.url, e.ticketsurl,
    e.free, e.permanent, e.modality, e.relatedurls, e.urlnames, e.images, e.videos, e.email, e.phone,
    e.activityurl, e.startdate, e.enddate, e.place, e.latitude, e.longitude, e.postalcode, e.municipality,
    e.region, e.address, oe.code, ue.creator, ue.datecreated,
    (
        CASE WHEN e.title ILIKE '%' || sqlc.arg(query)::text || '%' THEN 10 ELSE 0 END +
        CASE WHEN e.place ILIKE '%' || sqlc.arg(query)::text || '%' THEN 6 ELSE 0 END +
        CASE WHEN e.municipality ILIKE '%' || sqlc.arg(query)::text || '%' THEN 5 ELSE 0 END +
        CASE WHEN e.description ILIKE '%' || sqlc.arg(query)::text || '%' THEN 2 ELSE 0 END
    )::integer AS relevance
    FROM events e
    LEFT JOIN officialevents oe ON e.id = oe.id
    LEFT JOIN unofficialevents ue ON e.id = ue.id
    WHERE (
        e.title ILIKE '%' || sqlc.arg(query)::text || '%'
        OR e.place ILIKE '%' || sqlc.arg(query)::text || '%'
        OR e.municipality ILIKE '%' || sqlc.arg(query)::text || '%'
        OR e.description ILIKE '%' || sqlc.arg(query)::text || '%'
    )
    ORDER BY relevance DESC, e.startdate ASC NULLS LAST, e.id ASC
    LIMIT $1 OFFSET $2
) evlst
LEFT JOIN eventcategories ec ON evlst.id = ec.event
LEFT JOIN categories c ON ec.category = c.id;

-- name: CountSearchEvents :one
SELECT COUNT(*)::bigint
FROM events e
WHERE (
    e.title ILIKE '%' || sqlc.arg(query)::text || '%'
    OR e.place ILIKE '%' || sqlc.arg(query)::text || '%'
    OR e.municipality ILIKE '%' || sqlc.arg(query)::text || '%'
    OR e.description ILIKE '%' || sqlc.arg(query)::text || '%'
);
