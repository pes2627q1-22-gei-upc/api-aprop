-- name: GetAreaCongestionStats :one
-- In-person events within radius_km of (lat, lng) whose dates overlap [from_date, to_date], and the
-- number of their sign ups not marked as absent (expected attendance).
-- A non permanent event without end date lasts one day; a permanent one lasts until further notice.
WITH area_events AS (
    SELECT e.ID
    FROM Events e
    WHERE e.Latitude IS NOT NULL
      AND e.Longitude IS NOT NULL
      AND NOT (e.Latitude = 0 AND e.Longitude = 0)
      -- Bounding box prefilter: 1 degree of latitude is about 111 km
      AND e.Latitude BETWEEN sqlc.arg(lat)::float8 - sqlc.arg(radius_km)::float8 / 111.0
                         AND sqlc.arg(lat)::float8 + sqlc.arg(radius_km)::float8 / 111.0
      AND e.Longitude BETWEEN sqlc.arg(lng)::float8 - sqlc.arg(radius_km)::float8 / (111.0 * GREATEST(cos(radians(sqlc.arg(lat)::float8)), 0.01))
                          AND sqlc.arg(lng)::float8 + sqlc.arg(radius_km)::float8 / (111.0 * GREATEST(cos(radians(sqlc.arg(lat)::float8)), 0.01))
      -- Haversine distance in km
      AND 2 * 6371.0 * asin(LEAST(1.0, sqrt(
            power(sin(radians(e.Latitude - sqlc.arg(lat)::float8) / 2), 2)
            + cos(radians(sqlc.arg(lat)::float8)) * cos(radians(e.Latitude))
              * power(sin(radians(e.Longitude - sqlc.arg(lng)::float8) / 2), 2)
          ))) <= sqlc.arg(radius_km)::float8
      AND COALESCE(e.Modality::text, '') <> 'Online'
      AND e.StartDate IS NOT NULL
      AND e.StartDate <= sqlc.arg(to_date)::date
      AND COALESCE(e.EndDate, CASE WHEN e.Permanent THEN 'infinity'::date ELSE e.StartDate END) >= sqlc.arg(from_date)::date
)
SELECT
    COUNT(*)::int AS events_count,
    COALESCE(SUM(s.signups), 0)::int AS expected_attendance
FROM area_events ae
LEFT JOIN (
    SELECT su.Event, COUNT(*) AS signups
    FROM SignUp su
    WHERE su.Attendance IS DISTINCT FROM false
    GROUP BY su.Event
) s ON s.Event = ae.ID;
