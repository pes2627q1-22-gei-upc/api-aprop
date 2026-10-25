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

