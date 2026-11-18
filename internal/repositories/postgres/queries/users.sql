-- name: GetUserByUsername :one
SELECT u.id, u.firebaseuid, u.username, u.name, u.email, u.profilepicture, u.biography,
       u.dateregistered, u.accountstate, u.experience, u.level, u.rating, u.settings
FROM Users u
WHERE LOWER(u.Username) = LOWER(sqlc.arg(username)::varchar);

-- name: GetUserByFirebaseUIDWithSettings :one
SELECT u.id, u.firebaseuid, u.username, u.name, u.email, u.profilepicture, u.biography,
       u.dateregistered, u.accountstate, u.experience, u.level, u.rating, u.settings
FROM Users u
WHERE u.FirebaseUID = sqlc.arg(firebase_uid)::varchar;

-- name: GetUserByUUID :one
SELECT u.id, u.firebaseuid, u.username, u.name, u.email, u.profilepicture, u.biography,
       u.dateregistered, u.accountstate, u.experience, u.level, u.rating, u.settings
FROM Users u
WHERE u.id = sqlc.arg(uuid)::uuid;

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
-- Mean of the ratings of every NoOficial event created by the user; count = 0 means "no ratings".
SELECT COALESCE(AVG(er.Rating), 0)::real AS reliability, COUNT(er.Rating)::bigint AS ratings_count
FROM EventRatings er
JOIN UnofficialEvents ue ON ue.ID = er.Event
WHERE ue.Creator = sqlc.arg(creator)::uuid;

-- name: ListUserReceivedRatings :many
SELECT er.Event AS event_id, e.Title AS event_title, er.Rating, er.RatingDate
FROM EventRatings er
JOIN UnofficialEvents ue ON ue.ID = er.Event
JOIN Events e ON e.ID = er.Event
WHERE ue.Creator = sqlc.arg(creator)::uuid
ORDER BY er.RatingDate DESC, er.Event DESC;

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
WHERE ID = sqlc.arg(id)::uuid
RETURNING *;

-- name: UpdateUserSettings :one
UPDATE Settings SET
    ShowPlans = sqlc.arg(show_plans)::boolean,
    NotificationBefore = (sqlc.arg(notification_before)::varchar)::interval
WHERE ID = sqlc.arg(id)::integer
RETURNING *;

-- name: GetUserFriends :many
SELECT user2 from friendships where user1=$1;

-- name: GetFriendRequests :many
SELECT * from friendrequests where state='Pending'::friendrequeststate and receiver=$1 ORDER BY datesent DESC;

-- name: IsBlocked :one
SELECT EXISTS (SELECT 1 from blocked where (blockeduser = $1 and blockinguser = $2) or (blockeduser = $2 and blockinguser = $1) limit 1);

-- name: PendingRequestExists :one
SELECT EXISTS (SELECT 1 from friendrequests where state = 'Pending'::friendrequeststate and ((sender=$1 and receiver=$2) or (sender=$2 and receiver=$1)) limit 1);

-- name: CreateFriendRequest :one
INSERT INTO friendrequests (sender, receiver, datesent) VALUES ($1, $2, now()) returning *;

-- name: AcceptFriendRequest :one
UPDATE friendrequests SET state='Accepted'::friendrequeststate, dateresolved=now() where sender=$1 and receiver=$2 and datesent=$3 and state='Pending'::friendrequeststate RETURNING *;

-- name: AddFriendship :exec
INSERT INTO friendships (user1, user2) values ($1, $2), ($2, $1);

-- name: DenyFriendRequest :one
UPDATE friendrequests SET state='Rejected'::friendrequeststate, dateresolved=now() where sender=$1 and receiver=$2 and datesent=$3 and state='Pending'::friendrequeststate RETURNING *;

-- name: RemoveFriendship :exec
DELETE from friendships where (user1=$1 and user2=$2) or (user1=$2 and user2=$1);

-- name: GetSettings :one
SELECT * from settings where id=$1;