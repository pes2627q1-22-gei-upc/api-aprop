// Code generated for event search queries. DO NOT EDIT manually.

package sqlc

import (
	"context"
)

type SearchEventsWithRelevanceParams struct {
	Query  string
	Limit  int32
	Offset int32
}

const searchEventsWithRelevance = `-- name: SearchEventsWithRelevance :many
SELECT evlst.id, evlst.title, evlst.subtitle, evlst.description, evlst.schedule, evlst.tickets, evlst.url, evlst.ticketsurl,
       evlst.free, evlst.permanent, evlst.modality, evlst.relatedurls, evlst.urlnames, evlst.images, evlst.videos,
       evlst.email, evlst.phone, evlst.activityurl, evlst.startdate, evlst.enddate, evlst.place, evlst.latitude,
       evlst.longitude, evlst.postalcode, evlst.municipality, evlst.region, evlst.address, evlst.code, evlst.creator,
       evlst.datecreated, c.name as category FROM (
    SELECT e.id, e.title, e.subtitle, e.description, e.schedule, e.tickets, e.url, e.ticketsurl,
    e.free, e.permanent, e.modality, e.relatedurls, e.urlnames, e.images, e.videos, e.email, e.phone,
    e.activityurl, e.startdate, e.enddate, e.place, e.latitude, e.longitude, e.postalcode, e.municipality,
    e.region, e.address, oe.code, ue.creator, ue.datecreated,
    (
        CASE WHEN e.title ILIKE '%' || $1::text || '%' THEN 10 ELSE 0 END +
        CASE WHEN e.place ILIKE '%' || $1::text || '%' THEN 6 ELSE 0 END +
        CASE WHEN e.municipality ILIKE '%' || $1::text || '%' THEN 5 ELSE 0 END +
        CASE WHEN e.description ILIKE '%' || $1::text || '%' THEN 2 ELSE 0 END
    )::integer AS relevance
    FROM events e
    LEFT JOIN officialevents oe ON e.id = oe.id
    LEFT JOIN unofficialevents ue ON e.id = ue.id
    WHERE (
        e.title ILIKE '%' || $1::text || '%'
        OR e.place ILIKE '%' || $1::text || '%'
        OR e.municipality ILIKE '%' || $1::text || '%'
        OR e.description ILIKE '%' || $1::text || '%'
    )
    ORDER BY relevance DESC, e.startdate ASC NULLS LAST, e.id ASC
    LIMIT $2 OFFSET $3
) evlst
LEFT JOIN eventcategories ec ON evlst.id = ec.event
LEFT JOIN categories c ON ec.category = c.id
`

func (q *Queries) SearchEventsWithRelevance(ctx context.Context, arg SearchEventsWithRelevanceParams) ([]ListEventsRow, error) {
	rows, err := q.db.Query(ctx, searchEventsWithRelevance, arg.Query, arg.Limit, arg.Offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []ListEventsRow
	for rows.Next() {
		var i ListEventsRow
		if err := rows.Scan(
			&i.ID,
			&i.Title,
			&i.Subtitle,
			&i.Description,
			&i.Schedule,
			&i.Tickets,
			&i.Url,
			&i.Ticketsurl,
			&i.Free,
			&i.Permanent,
			&i.Modality,
			&i.Relatedurls,
			&i.Urlnames,
			&i.Images,
			&i.Videos,
			&i.Email,
			&i.Phone,
			&i.Activityurl,
			&i.Startdate,
			&i.Enddate,
			&i.Place,
			&i.Latitude,
			&i.Longitude,
			&i.PostalCode,
			&i.Municipality,
			&i.Region,
			&i.Address,
			&i.Code,
			&i.Creator,
			&i.Datecreated,
			&i.Category,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const countSearchEvents = `-- name: CountSearchEvents :one
SELECT COUNT(*)::bigint
FROM events e
WHERE (
    e.title ILIKE '%' || $1::text || '%'
    OR e.place ILIKE '%' || $1::text || '%'
    OR e.municipality ILIKE '%' || $1::text || '%'
    OR e.description ILIKE '%' || $1::text || '%'
)
`

func (q *Queries) CountSearchEvents(ctx context.Context, query string) (int64, error) {
	row := q.db.QueryRow(ctx, countSearchEvents, query)
	var count int64
	err := row.Scan(&count)
	return count, err
}
