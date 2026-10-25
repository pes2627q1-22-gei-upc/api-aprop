-- TODO(DP-024): replicar a supabase/migrations/ quan hi siguin 000006, 000008 i 000009.
-- Users.ID is already a UUID (000009), so the bridge UUID column from 000007 is redundant.
ALTER TABLE Users DROP COLUMN IF EXISTS UUID;
