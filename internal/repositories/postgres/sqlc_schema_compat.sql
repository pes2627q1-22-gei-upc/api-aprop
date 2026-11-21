-- Compatibility declarations for sqlc's static schema parser.
-- migrations/000018_unofficial_event_state.up.sql wraps CREATE TYPE inside a
-- PL/pgSQL DO $$ block (for idempotency), which sqlc skips, causing
-- migrations/000020_event_cancellation.up.sql (ALTER TYPE ...) to fail.
-- Keep this file in sync if future migrations create types inside DO $$ blocks.
CREATE TYPE UnofficialEventState AS ENUM ('Active', 'Withdrawn');
