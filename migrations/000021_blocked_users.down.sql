DROP INDEX IF EXISTS blocked_blockeduser_idx;
ALTER TABLE Blocked DROP COLUMN IF EXISTS DateBlocked;
