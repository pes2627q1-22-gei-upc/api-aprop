-- name: ListMapEvents :many
-- Events with valid coordinates inside the [min_lat, max_lat] x [min_lng, max_lng] bounding box,
-- enriched with official status and whether any friend of user_id is signed up or created the event.
SELECT
    e.id, e.title, e.subtitle, e.description, e.schedule, e.url, e.tickets, e.ticketsurl,
    e.free, e.permanent, COALESCE(e.modality::text, '')::text AS modality, e.relatedurls, e.urlnames, e.images,
    e.videos, e.email, e.place, e.latitude, e.longitude, e.postalcode, e.municipality,
    e.region, e.address, e.phone, e.activityurl, e.startdate, e.enddate, c.name as category,
    (ue.id is null)::boolean AS is_official, ue.creator, ue.datecreated, ue.state,
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
                SELECT 1 FROM Friendships f WHERE (f.User1 = sqlc.narg(user_id)::uuid AND f.User2 = ue.Creator)
                                            OR (f.User2 = sqlc.narg(user_id)::uuid AND f.User1 = ue.Creator)
            )
        )
    )::boolean AS has_friends
FROM Events e left join eventcategories ec on ec.event = e.ID left join categories c on ec.category = c.id left join unofficialevents ue on ue.id=e.id
WHERE (ue.id is null or ue.state='Active'::unofficialeventstate)
  AND e.Latitude IS NOT NULL
  AND e.Longitude IS NOT NULL
  AND NOT (e.Latitude = 0 AND e.Longitude = 0)
  AND e.Latitude >= sqlc.arg(min_lat)::real
  AND e.Latitude <= sqlc.arg(max_lat)::real
  AND e.Longitude >= sqlc.arg(min_lng)::real
  AND e.Longitude <= sqlc.arg(max_lng)::real
  AND e.enddate is null or e.enddate > now()
ORDER BY e.StartDate NULLS LAST, e.ID;
