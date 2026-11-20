-- Bloqueig (Usuari bloqueja Usuari). The table exists since 000001; this adds when the block was made, to list the
-- blocked users from the most recent one, and an index for the lookups by blocked user (IsBlocked, SearchUsers).
ALTER TABLE Blocked ADD COLUMN IF NOT EXISTS DateBlocked Timestamp NOT NULL DEFAULT now();
CREATE INDEX IF NOT EXISTS blocked_blockeduser_idx ON Blocked (BlockedUser);
