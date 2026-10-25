-- name: GetUserByID :one
SELECT u.id, u.firebaseuid, u.username, u.name, u.email, u.profilepicture, u.biography,
       u.dateregistered, u.accountstate, u.experience, u.level, u.rating, u.settings, u.uuid,
       COALESCE(s.showplans, true)::boolean AS show_plans
FROM Users u
LEFT JOIN Settings s ON s.ID = u.Settings
WHERE u.ID = sqlc.arg(id)::integer;

-- name: GetUserByUsername :one
SELECT u.id, u.firebaseuid, u.username, u.name, u.email, u.profilepicture, u.biography,
       u.dateregistered, u.accountstate, u.experience, u.level, u.rating, u.settings, u.uuid,
       COALESCE(s.showplans, true)::boolean AS show_plans
FROM Users u
LEFT JOIN Settings s ON s.ID = u.Settings
WHERE LOWER(u.Username) = LOWER(sqlc.arg(username)::varchar);

-- name: GetUserByFirebaseUIDWithSettings :one
SELECT u.id, u.firebaseuid, u.username, u.name, u.email, u.profilepicture, u.biography,
       u.dateregistered, u.accountstate, u.experience, u.level, u.rating, u.settings, u.uuid,
       COALESCE(s.showplans, true)::boolean AS show_plans
FROM Users u
LEFT JOIN Settings s ON s.ID = u.Settings
WHERE u.FirebaseUID = sqlc.arg(firebase_uid)::varchar;

-- name: GetUserByUUID :one
SELECT u.id, u.firebaseuid, u.username, u.name, u.email, u.profilepicture, u.biography,
       u.dateregistered, u.accountstate, u.experience, u.level, u.rating, u.settings, u.uuid,
       COALESCE(s.showplans, true)::boolean AS show_plans
FROM Users u
LEFT JOIN Settings s ON s.ID = u.Settings
WHERE u.UUID = sqlc.arg(uuid)::uuid;

-- name: CountFriends :one
SELECT COUNT(*)::bigint
FROM Friendships
WHERE User1 = sqlc.arg(user_id)::uuid OR User2 = sqlc.arg(user_id)::uuid;

-- name: AreFriends :one
SELECT EXISTS (
    SELECT 1 FROM Friendships
    WHERE (User1 = sqlc.arg(user1)::uuid AND User2 = sqlc.arg(user2)::uuid)
       OR (User1 = sqlc.arg(user2)::uuid AND User2 = sqlc.arg(user1)::uuid)
)::boolean;

-- name: CountUserPlans :one
SELECT COUNT(*)::bigint
FROM SignUp
WHERE UserID = sqlc.arg(user_id)::uuid;

-- name: CountCompletedUserPlans :one
SELECT COUNT(*)::bigint
FROM SignUp s
JOIN Events e ON e.ID = s.Event
WHERE s.UserID = sqlc.arg(user_id)::uuid
  AND (e.EndDate < NOW() OR (e.EndDate IS NULL AND e.StartDate < NOW()));

-- name: GetUserReliability :one
SELECT AVG(er.Rating)::real AS reliability
FROM EventRatings er
JOIN UnofficialEvents ue ON ue.ID = er.Event
WHERE ue.Creator = sqlc.arg(creator)::uuid;

-- name: ListUserCreatedEvents :many
SELECT e.id, e.title, e.subtitle, e.description, e.schedule, e.tickets, e.url, e.ticketsurl,
e.free, e.permanent, e.modality, e.relatedurls, e.urlnames, e.images, e.videos, e.email, e.phone,
e.activityurl, e.startdate, e.enddate, e.place, e.latitude, e.longitude, e.postalcode, e.municipality,
e.region, e.address, ue.creator, ue.datecreated
FROM events e
JOIN unofficialevents ue ON e.id = ue.id
WHERE ue.creator = sqlc.arg(creator)::uuid
ORDER BY e.startdate DESC NULLS LAST, e.id DESC;

-- name: UpdateUserProfile :one
UPDATE Users SET
    Name = COALESCE(sqlc.narg(name)::varchar, Name),
    Username = COALESCE(sqlc.narg(username)::varchar, Username),
    Biography = COALESCE(sqlc.narg(biography)::varchar, Biography),
    ProfilePicture = COALESCE(sqlc.narg(profile_picture)::varchar, ProfilePicture)
WHERE ID = sqlc.arg(id)::integer
RETURNING *;

-- name: UpdateUserSettings :one
UPDATE Settings SET
    ShowPlans = sqlc.arg(show_plans)::boolean
WHERE ID = sqlc.arg(id)::integer
RETURNING *;
