ALTER TABLE Settings DROP COLUMN IF EXISTS PushEnabled;
DROP INDEX IF EXISTS notifications_pending_push_idx;
ALTER TABLE Notifications DROP COLUMN IF EXISTS PushedAt;
DROP TABLE IF EXISTS DeviceTokens;
