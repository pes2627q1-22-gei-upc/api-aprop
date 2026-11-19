-- Chat of an event (Missatge). The table already existed as Messages; it is renamed to follow the naming of the other
-- event tables (EventRatings). The primary key (UserID, Event, DateSent) is the R1 key author + event + date sent.
ALTER TABLE IF EXISTS Messages RENAME TO EventMessages;
ALTER TABLE EventMessages ALTER COLUMN DateSent SET DEFAULT now();
ALTER TABLE EventMessages DROP CONSTRAINT IF EXISTS eventmessages_message_length;
ALTER TABLE EventMessages ADD CONSTRAINT eventmessages_message_length
    CHECK (char_length(btrim(Message)) BETWEEN 1 AND 1000);
CREATE INDEX IF NOT EXISTS eventmessages_event_datesent_idx ON EventMessages (Event, DateSent DESC);
