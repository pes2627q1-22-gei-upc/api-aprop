-- name: ListMapEvents :many
-- Events with valid coordinates inside the [min_lat, max_lat] x [min_lng, max_lng] bounding box,
-- enriched with official status and whether any friend of user_id is signed up or created the event.
SELECT
    e.id, e.title, e.subtitle, e.description, e.schedule, e.url, e.tickets, e.ticketsurl,
    e.free, e.permanent, COALESCE(e.modality::text, '')::text AS modality, e.relatedurls, e.urlnames, e.images,
    e.videos, e.email, e.place, e.latitude, e.longitude, e.postalcode, e.municipality,
    e.region, e.address, e.phone, e.activityurl, e.startdate, e.enddate,
    (
        EXISTS (SELECT 1 FROM OfficialEvents oe WHERE oe.ID = e.ID)
        OR (
            NOT EXISTS (SELECT 1 FROM UnofficialEvents ue WHERE ue.ID = e.ID)
            AND NOT EXISTS (SELECT 1 FROM OfficialEvents)
        )
    )::boolean AS is_official,
    (
        sqlc.narg(user_id)::uuid IS NOT NULL
        AND (
            EXISTS (
                SELECT 1 FROM SignUp s
                JOIN Friendships f ON (
                    (f.User1 = sqlc.narg(user_id)::uuid AND f.User2 = s.UserID)
                    OR (f.User2 = sqlc.narg(user_id)::uuid AND f.User1 = s.UserID)
                )
                WHERE s.Event = e.ID
            )
            OR EXISTS (
                SELECT 1 FROM UnofficialEvents ue
                JOIN Friendships f ON (
                    (f.User1 = sqlc.narg(user_id)::uuid AND f.User2 = ue.Creator)
                    OR (f.User2 = sqlc.narg(user_id)::uuid AND f.User1 = ue.Creator)
                )
                WHERE ue.ID = e.ID
            )
        )
    )::boolean AS has_friends
FROM Events e
WHERE e.Latitude IS NOT NULL
  AND e.Longitude IS NOT NULL
  AND NOT (e.Latitude = 0 AND e.Longitude = 0)
  AND e.Latitude >= sqlc.arg(min_lat)::real
  AND e.Latitude <= sqlc.arg(max_lat)::real
  AND e.Longitude >= sqlc.arg(min_lng)::real
  AND e.Longitude <= sqlc.arg(max_lng)::real
ORDER BY e.StartDate NULLS LAST, e.ID;
