-- Chat of an event (Missatge). The table already existed as Messages; it is renamed to follow the naming of the other
-- event tables (EventRatings). The primary key (UserID, Event, DateSent) is the R1 key author + event + date sent.
ALTER TABLE IF EXISTS Messages RENAME TO EventMessages;
ALTER TABLE EventMessages ALTER COLUMN DateSent SET DEFAULT now();
-- Message holds a format byte followed by the text: 0 = plain UTF-8, 1 = DEFLATE compressed (DP-040). The API compresses
-- it because Postgres only compresses rows over ~2 kB. Existing rows become plain. The text length (1..1000 characters)
-- is checked by the API; the database only bounds the size: format byte + 1..4000 bytes.
ALTER TABLE EventMessages ALTER COLUMN Message TYPE bytea USING CASE
    WHEN pg_typeof(Message) = 'bytea'::regtype THEN Message::text::bytea
    ELSE '\x00'::bytea || convert_to(Message::text, 'UTF8')
END;
ALTER TABLE EventMessages DROP CONSTRAINT IF EXISTS eventmessages_message_size;
ALTER TABLE EventMessages ADD CONSTRAINT eventmessages_message_size
    CHECK (octet_length(Message) BETWEEN 2 AND 4001);
CREATE INDEX IF NOT EXISTS eventmessages_event_datesent_idx ON EventMessages (Event, DateSent DESC);
