-- name: GetNotification :one
SELECT * FROM notifications where id = $1;

-- name: GetUserNotifications :many
SELECT * FROM notifications where userid = $1
and (sqlc.arg(filtertype)::integer = 0 or type = sqlc.arg(filtertype)::integer)
ORDER BY notificationdate desc;

-- name: CreateNotification :one
INSERT INTO notifications (userid, notificationdate, friend, requestdate, event, type, text) VALUES ($1, now(), $2, $3, $4, $5, $6) RETURNING id;

-- name: ReadNotification :exec
UPDATE notifications SET read = true where id = $1;

-- name: ReadUserNotifications :exec
UPDATE notifications SET read = true where userid = $1;

-- name: GetNotificationTypes :many
SELECT * FROM notificationtypes;

-- name: GetNotificationTypeByName :one
SELECT id FROM notificationtypes where name = $1;

-- name: RemoveNotificationTypes :exec
DELETE FROM notificationtypes;

-- name: RestoreNotificationTypes :exec
INSERT INTO notificationtypes (name) values
    ('Social'),
    ('Recomanacio'),
    ('Recordatori');

-- name: CreateReminders :many
INSERT INTO notifications (userid, notificationdate, type, event, text)
SELECT su.userid, now(), (SELECT id from notificationtypes where name='Recordatori'), e.id, CONCAT(e.title, ' comença aviat. No hi faltis!')
from signup su join events e on e.id=su.event join users u on u.id=su.userid join settings st on st.id=u.settings
left join notifications n on (n.event=e.id and n.userid=su.userid)
where n.id is null and e.startdate - now() > '0 seconds'::interval and e.startdate - now() < st.notificationbefore
returning id;

-- name: CreateFriendSignupNotifications :many
INSERT INTO notifications (userid, notificationdate, type, event, friend, text)
SELECT f.user2, now(), (select id from notificationtypes where name='Social'), @event::integer, @userid::uuid,
CONCAT((select username from users where id = @userid::uuid), ' s''ha ', @signuptext::varchar, ' ', (select title from events where id = @event::integer))
FROM friendships f join users u on f.user1=u.id join settings s on u.settings=s.id
WHERE f.user1 = @userid::uuid and s.showplans
RETURNING id;