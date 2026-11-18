-- name: ListRecommendedEvents :many
-- Active events not yet joined by the user, ranked globally in SQL by UserPreferences (Interes./pes),
-- user SignUp category affinity and global sign-up popularity, paginated before fetching event details.
WITH user_signup_cats AS (
    SELECT ec.category, COUNT(DISTINCT s.event)::double precision AS signup_count
    FROM signup s
    JOIN eventcategories ec ON ec.event = s.event
    WHERE s.userid = sqlc.arg(user_id)::uuid
    GROUP BY ec.category
),
active_candidates AS (
    SELECT e.id, e.enddate, COUNT(DISTINCT s.userid)::bigint AS num_signups
    FROM events e
    LEFT JOIN signup s ON s.event = e.id
    WHERE (e.enddate IS NULL OR e.enddate > now())
      AND NOT EXISTS (
          SELECT 1 FROM signup us
          WHERE us.event = e.id AND us.userid = sqlc.arg(user_id)::uuid
      )
    GROUP BY e.id, e.enddate
),
paged_events AS (
    SELECT
        ac.id,
        ac.num_signups,
        (
            COALESCE(
                AVG(
                    COALESCE(up.weight::double precision, 1.0) +
                    LEAST(COALESCE(usc.signup_count, 0.0) * 0.5, 2.0)
                ),
                1.0
            ) +
            CASE
                WHEN ac.num_signups > 0
                THEN LEAST(0.1 * LN(1.0 + ac.num_signups::double precision), 0.5)
                ELSE 0.0
            END
        )::double precision AS score,
        ac.enddate
    FROM active_candidates ac
    LEFT JOIN eventcategories ec ON ec.event = ac.id
    LEFT JOIN userpreferences up ON up.userid = sqlc.arg(user_id)::uuid AND up.category = ec.category
    LEFT JOIN user_signup_cats usc ON usc.category = ec.category
    GROUP BY ac.id, ac.enddate, ac.num_signups
    ORDER BY score DESC, ac.num_signups DESC, ac.enddate ASC NULLS LAST, ac.id ASC
    LIMIT sqlc.arg(page_size)::int OFFSET sqlc.arg(page_offset)::int
)
SELECT
    e.id, e.title, e.subtitle, e.description, e.schedule, e.url, e.tickets, e.ticketsurl,
    e.free, e.permanent, e.modality, e.relatedurls, e.urlnames, e.images, e.videos, e.email, e.phone,
    e.activityurl, e.startdate, e.enddate, e.place, e.latitude, e.longitude, e.postalcode, e.municipality,
    e.region, e.address, oe.code, ue.creator, ue.datecreated,
    pe.num_signups, pe.score, c.name AS category
FROM paged_events pe
JOIN events e ON e.id = pe.id
LEFT JOIN officialevents oe ON e.id = oe.id
LEFT JOIN unofficialevents ue ON e.id = ue.id
LEFT JOIN eventcategories ec ON e.id = ec.event
LEFT JOIN categories c ON ec.category = c.id
ORDER BY pe.score DESC, pe.num_signups DESC, pe.enddate ASC NULLS LAST, pe.id ASC;
