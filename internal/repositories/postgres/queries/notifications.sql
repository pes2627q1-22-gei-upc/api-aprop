-- name: GetNotification :one
SELECT * FROM notifications where id = $1;

-- name: GetUserNotifications :many
SELECT * FROM notifications where userid = $1
and (sqlc.arg(filtertype)::integer = 0 or type = sqlc.arg(filtertype)::integer);

-- name: CreateNotification :one
INSERT INTO notifications (userid, notificationdate, friend, requestdate, event, type) VALUES ($1, $2, $3, $4, $5, $6) RETURNING id;

-- name: ReadNotification :exec
UPDATE notifications SET read = true where id = $1;

-- name: ReadUserNotifications :exec
UPDATE notifications SET read = true where userid = $1;

-- name: GetNotificationTypes :many
SELECT * FROM notificationtypes;

-- name: RemoveNotificationTypes :exec
DELETE FROM notificationtypes;

-- name: RestoreNotificationTypes :exec
INSERT INTO notificationtypes (name) values
    ('Social'),
    ('Recomanacio'),
    ('Recordatori');

-- name: CreateReminders :many
INSERT INTO notifications (userid, notificationdate, type, event)
SELECT su.userid, now(), (SELECT id from notificationtypes where name='Recordatori'), e.id
from signup su join events e on e.id=su.event join users u on u.id=su.userid join settings st on st.id=u.settings
left join notifications n on (n.event=e.id and n.userid=su.userid)
where n.id is null and e.startdate - now() > '0 seconds'::interval and e.startdate - now() < st.notificationbefore
returning id;