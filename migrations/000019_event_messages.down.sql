DROP INDEX IF EXISTS eventmessages_event_datesent_idx;
ALTER TABLE EventMessages DROP CONSTRAINT IF EXISTS eventmessages_message_length;
ALTER TABLE EventMessages ALTER COLUMN DateSent DROP DEFAULT;
ALTER TABLE EventMessages RENAME TO Messages;
