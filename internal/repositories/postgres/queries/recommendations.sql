-- name: ListActiveEventsForRecommendations :many
-- Active events (no end date or end date in the future) with their distinct sign-up count and categories.
SELECT evlst.*, c.name AS category
FROM (
    SELECT
        e.id, e.title, e.subtitle, e.description, e.schedule, e.url, e.tickets, e.ticketsurl,
        e.free, e.permanent, e.modality, e.relatedurls, e.urlnames, e.images, e.videos, e.email, e.phone,
        e.activityurl, e.startdate, e.enddate, e.place, e.latitude, e.longitude, e.postalcode, e.municipality,
        e.region, e.address, oe.code, ue.creator, ue.datecreated,
        COUNT(DISTINCT s.userid)::bigint AS num_signups
    FROM events e
    LEFT JOIN officialevents oe ON e.id = oe.id
    LEFT JOIN unofficialevents ue ON e.id = ue.id
    LEFT JOIN signup s ON s.event = e.id
    WHERE e.enddate IS NULL OR e.enddate > now()
    GROUP BY e.id, oe.id, ue.id
) evlst
LEFT JOIN eventcategories ec ON evlst.id = ec.event
LEFT JOIN categories c ON ec.category = c.id
ORDER BY evlst.enddate ASC NULLS LAST, evlst.id ASC;
