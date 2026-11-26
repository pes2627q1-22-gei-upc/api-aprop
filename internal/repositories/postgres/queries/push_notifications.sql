-- Push notifications (Taiga #275, DP-050): device tokens and the outbox of Notifications.

-- name: UpsertDeviceToken :exec
-- A token identifies one app install: registering it again moves it to the current user.
INSERT INTO DeviceTokens (Token, UserID, Platform, UpdatedAt)
VALUES (sqlc.arg(token)::varchar, sqlc.arg(user_id)::uuid, sqlc.arg(platform)::varchar, now())
ON CONFLICT (Token) DO UPDATE SET
    UserID = EXCLUDED.UserID,
    Platform = EXCLUDED.Platform,
    UpdatedAt = EXCLUDED.UpdatedAt;

-- name: DeleteUserDeviceToken :execrows
DELETE FROM DeviceTokens WHERE Token = sqlc.arg(token)::varchar AND UserID = sqlc.arg(user_id)::uuid;

-- name: DeleteDeviceTokens :exec
DELETE FROM DeviceTokens WHERE Token = ANY(sqlc.arg(tokens)::varchar[]);

-- name: GetDeviceTokensByUsers :many
SELECT UserID, Token FROM DeviceTokens WHERE UserID = ANY(sqlc.arg(user_ids)::uuid[]) ORDER BY UserID, UpdatedAt DESC;

-- name: ExpirePendingPushes :execrows
-- The pending pushes older than max_age are marked without sending them (an old reminder is no longer useful).
UPDATE Notifications SET PushedAt = now()
WHERE PushedAt IS NULL AND NotificationDate < now() - sqlc.arg(max_age)::interval;

-- name: ClaimPendingPushes :many
-- Locks a batch of pending pushes until the transaction ends; SKIP LOCKED lets several instances share the outbox.
SELECT n.ID, n.UserID, n.Event, n.Friend, n.Text, nt.Name AS TypeName, s.PushEnabled
FROM Notifications n
JOIN NotificationTypes nt ON nt.ID = n.Type
JOIN Users u ON u.ID = n.UserID
JOIN Settings s ON s.ID = u.Settings
WHERE n.PushedAt IS NULL
ORDER BY n.ID
LIMIT sqlc.arg(batch_size)::integer
FOR UPDATE OF n SKIP LOCKED;

-- name: MarkPushed :exec
UPDATE Notifications SET PushedAt = now() WHERE ID = ANY(sqlc.arg(ids)::integer[]);

-- name: GetSettingsPushEnabled :one
SELECT PushEnabled FROM Settings WHERE ID = sqlc.arg(id)::integer;

-- name: UpdateSettingsPushEnabled :exec
UPDATE Settings SET PushEnabled = sqlc.arg(push_enabled)::boolean WHERE ID = sqlc.arg(id)::integer;
