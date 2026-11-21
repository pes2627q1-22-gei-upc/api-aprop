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
DELETE FROM notificationtypes where name!='Social' and name!='Recomanacio' and name!='Recordatori' and name!='Cancelacio';

-- name: RestoreNotificationTypes :exec
INSERT INTO notificationtypes (name) values
    ('Social'),
    ('Recomanacio'),
    ('Recordatori'),
    ('Cancelacio') on conflict do nothing;

-- name: CreateReminders :many
WITH nottype AS (SELECT id from notificationtypes where name='Recordatori')
INSERT INTO notifications (userid, notificationdate, type, event, text)
SELECT su.userid, now(), nt.id, e.id, CONCAT(e.title, ' comença aviat. No hi faltis!')
from signup su join events e on e.id=su.event join users u on u.id=su.userid join settings st on st.id=u.settings cross join nottype nt
left join notifications n on (n.event=e.id and n.userid=su.userid)
where n.id is null and exists(select * from allowednotifications where settings=st.id and notificationtype=nt.id)
and e.startdate - now() > '0 seconds'::interval and e.startdate - now() < st.notificationbefore
returning id;

-- name: CreateFriendSignupNotifications :many
WITH nottype AS (SELECT id from notificationtypes where name='Social')
INSERT INTO notifications (userid, notificationdate, type, event, friend, text)
SELECT f.user2, now(), nt.id, @event::integer, @userid::uuid,
CONCAT((select username from users where id = @userid::uuid), ' s''ha ', @signuptext::varchar, ' ', (select title from events where id = @event::integer))
FROM friendships f join users u on f.user1=u.id join settings s on u.settings=s.id join users u2 on f.user2=u2.id join settings s2 on s2.id=u2.settings cross join nottype nt
WHERE exists(select * from allowednotifications where settings=s2.id and notificationtype=nt.id)
and f.user1 = @userid::uuid and s.showplans
RETURNING id;

-- name: CreateCancellationNotifications :execrows
-- Notifies once every user signed up to the cancelled event, except its creator (NotificacióEsdeveniment Cancel·lació).
WITH nottype as (SELECT id FROM notificationtypes WHERE name = 'Cancelacio')
INSERT INTO notifications (userid, notificationdate, type, event, text)
SELECT su.userid, now(), nt.id, e.id, CONCAT(e.title, ' s''ha cancel·lat.')
FROM (SELECT DISTINCT userid FROM signup WHERE event = sqlc.arg(event_id)::integer) su
JOIN events e ON e.id = sqlc.arg(event_id)::integer
JOIN unofficialevents ue ON ue.id = e.id
JOIN users u on su.userid = u.id
JOIN settings s on s.id = u.settings
CROSS JOIN nottype nt
WHERE su.userid <> ue.creator and exists(select * from allowednotifications where settings=s.id and notificationtype=nt.id);
