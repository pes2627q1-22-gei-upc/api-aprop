// Code generated for users and user profile queries. DO NOT EDIT manually unless needed.

package sqlc

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

type GetUserRow struct {
	ID             int32
	Firebaseuid    string
	Username       string
	Name           pgtype.Text
	Email          pgtype.Text
	Profilepicture pgtype.Text
	Biography      pgtype.Text
	Dateregistered pgtype.Timestamp
	Accountstate   int32
	Experience     int32
	Level          pgtype.Int4
	Rating         pgtype.Float4
	Settings       int32
	Uuid           pgtype.UUID
	ShowPlans      bool
}

const getUserByID = `-- name: GetUserByID :one
SELECT u.id, u.firebaseuid, u.username, u.name, u.email, u.profilepicture, u.biography,
       u.dateregistered, u.accountstate, u.experience, u.level, u.rating, u.settings, u.uuid,
       COALESCE(s.showplans, true)::boolean AS show_plans
FROM Users u
LEFT JOIN Settings s ON s.ID = u.Settings
WHERE u.ID = $1::integer
`

func (q *Queries) GetUserByID(ctx context.Context, id int32) (GetUserRow, error) {
	row := q.db.QueryRow(ctx, getUserByID, id)
	var i GetUserRow
	err := row.Scan(
		&i.ID,
		&i.Firebaseuid,
		&i.Username,
		&i.Name,
		&i.Email,
		&i.Profilepicture,
		&i.Biography,
		&i.Dateregistered,
		&i.Accountstate,
		&i.Experience,
		&i.Level,
		&i.Rating,
		&i.Settings,
		&i.Uuid,
		&i.ShowPlans,
	)
	return i, err
}

const getUserByUsername = `-- name: GetUserByUsername :one
SELECT u.id, u.firebaseuid, u.username, u.name, u.email, u.profilepicture, u.biography,
       u.dateregistered, u.accountstate, u.experience, u.level, u.rating, u.settings, u.uuid,
       COALESCE(s.showplans, true)::boolean AS show_plans
FROM Users u
LEFT JOIN Settings s ON s.ID = u.Settings
WHERE LOWER(u.Username) = LOWER($1::varchar)
`

func (q *Queries) GetUserByUsername(ctx context.Context, username string) (GetUserRow, error) {
	row := q.db.QueryRow(ctx, getUserByUsername, username)
	var i GetUserRow
	err := row.Scan(
		&i.ID,
		&i.Firebaseuid,
		&i.Username,
		&i.Name,
		&i.Email,
		&i.Profilepicture,
		&i.Biography,
		&i.Dateregistered,
		&i.Accountstate,
		&i.Experience,
		&i.Level,
		&i.Rating,
		&i.Settings,
		&i.Uuid,
		&i.ShowPlans,
	)
	return i, err
}

const getUserByFirebaseUIDWithSettings = `-- name: GetUserByFirebaseUIDWithSettings :one
SELECT u.id, u.firebaseuid, u.username, u.name, u.email, u.profilepicture, u.biography,
       u.dateregistered, u.accountstate, u.experience, u.level, u.rating, u.settings, u.uuid,
       COALESCE(s.showplans, true)::boolean AS show_plans
FROM Users u
LEFT JOIN Settings s ON s.ID = u.Settings
WHERE u.FirebaseUID = $1::varchar
`

func (q *Queries) GetUserByFirebaseUIDWithSettings(ctx context.Context, firebaseUID string) (GetUserRow, error) {
	row := q.db.QueryRow(ctx, getUserByFirebaseUIDWithSettings, firebaseUID)
	var i GetUserRow
	err := row.Scan(
		&i.ID,
		&i.Firebaseuid,
		&i.Username,
		&i.Name,
		&i.Email,
		&i.Profilepicture,
		&i.Biography,
		&i.Dateregistered,
		&i.Accountstate,
		&i.Experience,
		&i.Level,
		&i.Rating,
		&i.Settings,
		&i.Uuid,
		&i.ShowPlans,
	)
	return i, err
}

const getUserByUUID = `-- name: GetUserByUUID :one
SELECT u.id, u.firebaseuid, u.username, u.name, u.email, u.profilepicture, u.biography,
       u.dateregistered, u.accountstate, u.experience, u.level, u.rating, u.settings, u.uuid,
       COALESCE(s.showplans, true)::boolean AS show_plans
FROM Users u
LEFT JOIN Settings s ON s.ID = u.Settings
WHERE u.UUID = $1::uuid
`

func (q *Queries) GetUserByUUID(ctx context.Context, uuid pgtype.UUID) (GetUserRow, error) {
	row := q.db.QueryRow(ctx, getUserByUUID, uuid)
	var i GetUserRow
	err := row.Scan(
		&i.ID,
		&i.Firebaseuid,
		&i.Username,
		&i.Name,
		&i.Email,
		&i.Profilepicture,
		&i.Biography,
		&i.Dateregistered,
		&i.Accountstate,
		&i.Experience,
		&i.Level,
		&i.Rating,
		&i.Settings,
		&i.Uuid,
		&i.ShowPlans,
	)
	return i, err
}

const countFriends = `-- name: CountFriends :one
SELECT COUNT(*)::bigint
FROM Friendships
WHERE User1 = $1::uuid OR User2 = $1::uuid
`

func (q *Queries) CountFriends(ctx context.Context, userID pgtype.UUID) (int64, error) {
	row := q.db.QueryRow(ctx, countFriends, userID)
	var count int64
	err := row.Scan(&count)
	return count, err
}

type AreFriendsParams struct {
	User1 pgtype.UUID
	User2 pgtype.UUID
}

const areFriends = `-- name: AreFriends :one
SELECT EXISTS (
    SELECT 1 FROM Friendships
    WHERE (User1 = $1::uuid AND User2 = $2::uuid)
       OR (User1 = $2::uuid AND User2 = $1::uuid)
)::boolean
`

func (q *Queries) AreFriends(ctx context.Context, arg AreFriendsParams) (bool, error) {
	row := q.db.QueryRow(ctx, areFriends, arg.User1, arg.User2)
	var exists bool
	err := row.Scan(&exists)
	return exists, err
}

const countUserPlans = `-- name: CountUserPlans :one
SELECT COUNT(*)::bigint
FROM SignUp
WHERE UserID = $1::uuid
`

func (q *Queries) CountUserPlans(ctx context.Context, userID pgtype.UUID) (int64, error) {
	row := q.db.QueryRow(ctx, countUserPlans, userID)
	var count int64
	err := row.Scan(&count)
	return count, err
}

const countCompletedUserPlans = `-- name: CountCompletedUserPlans :one
SELECT COUNT(*)::bigint
FROM SignUp s
JOIN Events e ON e.ID = s.Event
WHERE s.UserID = $1::uuid
  AND (e.EndDate < NOW() OR (e.EndDate IS NULL AND e.StartDate < NOW()))
`

func (q *Queries) CountCompletedUserPlans(ctx context.Context, userID pgtype.UUID) (int64, error) {
	row := q.db.QueryRow(ctx, countCompletedUserPlans, userID)
	var count int64
	err := row.Scan(&count)
	return count, err
}

const getUserReliability = `-- name: GetUserReliability :one
SELECT AVG(er.Rating)::real AS reliability
FROM EventRatings er
JOIN UnofficialEvents ue ON ue.ID = er.Event
WHERE ue.Creator = $1::uuid
`

func (q *Queries) GetUserReliability(ctx context.Context, creator pgtype.UUID) (pgtype.Float4, error) {
	row := q.db.QueryRow(ctx, getUserReliability, creator)
	var reliability pgtype.Float4
	err := row.Scan(&reliability)
	return reliability, err
}

type ListUserCreatedEventsRow struct {
	ID           int32
	Title        string
	Subtitle     pgtype.Text
	Description  string
	Schedule     pgtype.Text
	Tickets      pgtype.Text
	Url          pgtype.Text
	Ticketsurl   pgtype.Text
	Free         pgtype.Bool
	Permanent    bool
	Modality     pgtype.Text
	Relatedurls  pgtype.Text
	Urlnames     pgtype.Text
	Images       pgtype.Text
	Videos       pgtype.Text
	Email        pgtype.Text
	Phone        pgtype.Text
	Activityurl  pgtype.Text
	Startdate    pgtype.Date
	Enddate      pgtype.Date
	Place        pgtype.Text
	Latitude     float32
	Longitude    float32
	Postalcode   pgtype.Text
	Municipality pgtype.Text
	Region       pgtype.Text
	Address      pgtype.Text
	Creator      pgtype.UUID
	Datecreated  pgtype.Timestamp
}

const listUserCreatedEvents = `-- name: ListUserCreatedEvents :many
SELECT e.id, e.title, e.subtitle, e.description, e.schedule, e.tickets, e.url, e.ticketsurl,
e.free, e.permanent, e.modality, e.relatedurls, e.urlnames, e.images, e.videos, e.email, e.phone,
e.activityurl, e.startdate, e.enddate, e.place, e.latitude, e.longitude, e.postalcode, e.municipality,
e.region, e.address, ue.creator, ue.datecreated
FROM events e
JOIN unofficialevents ue ON e.id = ue.id
WHERE ue.creator = $1::uuid
ORDER BY e.startdate DESC NULLS LAST, e.id DESC
`

func (q *Queries) ListUserCreatedEvents(ctx context.Context, creator pgtype.UUID) ([]ListUserCreatedEventsRow, error) {
	rows, err := q.db.Query(ctx, listUserCreatedEvents, creator)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []ListUserCreatedEventsRow
	for rows.Next() {
		var i ListUserCreatedEventsRow
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
			&i.Postalcode,
			&i.Municipality,
			&i.Region,
			&i.Address,
			&i.Creator,
			&i.Datecreated,
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

type UpdateUserProfileParams struct {
	ID             int32
	Name           pgtype.Text
	Username       pgtype.Text
	Biography      pgtype.Text
	ProfilePicture pgtype.Text
}

const updateUserProfile = `-- name: UpdateUserProfile :one
UPDATE Users SET
    Name = COALESCE($2::varchar, Name),
    Username = COALESCE($3::varchar, Username),
    Biography = COALESCE($4::varchar, Biography),
    ProfilePicture = COALESCE($5::varchar, ProfilePicture)
WHERE ID = $1::integer
RETURNING id, firebaseuid, username, name, email, profilepicture, biography, dateregistered, accountstate, experience, level, rating, settings, uuid
`

func (q *Queries) UpdateUserProfile(ctx context.Context, arg UpdateUserProfileParams) (User, error) {
	row := q.db.QueryRow(ctx, updateUserProfile,
		arg.ID,
		arg.Name,
		arg.Username,
		arg.Biography,
		arg.ProfilePicture,
	)
	var i User
	err := row.Scan(
		&i.ID,
		&i.Firebaseuid,
		&i.Username,
		&i.Name,
		&i.Email,
		&i.Profilepicture,
		&i.Biography,
		&i.Dateregistered,
		&i.Accountstate,
		&i.Experience,
		&i.Level,
		&i.Rating,
		&i.Settings,
		&i.Uuid,
	)
	return i, err
}

type UpdateUserSettingsParams struct {
	ID        int32
	ShowPlans bool
}

const updateUserSettings = `-- name: UpdateUserSettings :one
UPDATE Settings SET
    ShowPlans = $2::boolean
WHERE ID = $1::integer
RETURNING id, showplans
`

func (q *Queries) UpdateUserSettings(ctx context.Context, arg UpdateUserSettingsParams) (Setting, error) {
	row := q.db.QueryRow(ctx, updateUserSettings, arg.ID, arg.ShowPlans)
	var i Setting
	err := row.Scan(&i.ID, &i.Showplans)
	return i, err
}
