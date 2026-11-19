DROP INDEX IF EXISTS eventmessages_event_datesent_idx;
ALTER TABLE EventMessages DROP CONSTRAINT IF EXISTS eventmessages_message_size;
-- SQL cannot decompress the DEFLATE messages (format byte 1): they are lost and replaced by a placeholder.
ALTER TABLE EventMessages ALTER COLUMN Message TYPE varchar USING CASE
    WHEN get_byte(Message, 0) = 0 THEN convert_from(substring(Message FROM 2), 'UTF8')
    ELSE '[missatge comprimit]'
END;
ALTER TABLE EventMessages ALTER COLUMN DateSent DROP DEFAULT;
ALTER TABLE EventMessages RENAME TO Messages;
