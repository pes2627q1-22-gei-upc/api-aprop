-- Push notifications (Taiga #275, DP-046). Every notification of the inbox (Notifications) is also sent as a push to the
-- devices of its user through Firebase Cloud Messaging.
-- DeviceTokens: FCM registration tokens of the devices of each user. A token identifies one app install, so it is the
-- key: when another user signs in on the same device, the token moves to that user. The tokens go with the user (RGPD).
CREATE TABLE IF NOT EXISTS DeviceTokens (
    Token     Varchar PRIMARY KEY,
    UserID    UUID NOT NULL REFERENCES Users(ID) ON DELETE CASCADE,
    Platform  Varchar NOT NULL CHECK (Platform IN ('android', 'ios', 'web')),
    UpdatedAt Timestamp NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS devicetokens_userid_idx ON DeviceTokens (UserID);
ALTER TABLE DeviceTokens ENABLE ROW LEVEL SECURITY;

-- Outbox: a notification is pending to be pushed while PushedAt is null. The existing ones are marked as pushed, so the
-- whole history is not sent when this is deployed.
ALTER TABLE Notifications ADD COLUMN IF NOT EXISTS PushedAt Timestamp;
UPDATE Notifications SET PushedAt = now() WHERE PushedAt IS NULL;
CREATE INDEX IF NOT EXISTS notifications_pending_push_idx ON Notifications (ID) WHERE PushedAt IS NULL;

-- Global switch of the push notifications of a user. The notifications still reach the inbox when it is off.
ALTER TABLE Settings ADD COLUMN IF NOT EXISTS PushEnabled Boolean NOT NULL DEFAULT true;
