-- Language of the user interface and of the event texts (US #28). TODO(DP-043)
ALTER TABLE Settings ADD COLUMN IF NOT EXISTS Language Varchar NOT NULL DEFAULT 'ca';
ALTER TABLE Settings DROP CONSTRAINT IF EXISTS settings_language_check;
ALTER TABLE Settings ADD CONSTRAINT settings_language_check CHECK (Language IN ('ca', 'es', 'en'));

-- Cache of the machine translations of the free-text fields of an event (title, subtitle, description, schedule,
-- tickets and comma-separated link names). The originals are in Catalan.
-- SourceHash is the SHA-256 (hex) of those original fields: when the event changes, the hash no longer matches and
-- the row is translated again.
CREATE TABLE IF NOT EXISTS EventTranslations (
    Event Integer NOT NULL REFERENCES Events(ID) ON DELETE CASCADE,
    Lang Varchar NOT NULL CHECK (Lang IN ('es', 'en')),
    Title Varchar NOT NULL,
    Subtitle Varchar NOT NULL DEFAULT '',
    Description Varchar NOT NULL,
    Schedule Varchar NOT NULL DEFAULT '',
    Tickets Varchar NOT NULL DEFAULT '',
    URLNames Varchar NOT NULL DEFAULT '',
    SourceHash Varchar NOT NULL,
    UpdatedAt Timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (Event, Lang)
);
ALTER TABLE EventTranslations ADD COLUMN IF NOT EXISTS Subtitle Varchar NOT NULL DEFAULT '';
ALTER TABLE EventTranslations ADD COLUMN IF NOT EXISTS Schedule Varchar NOT NULL DEFAULT '';
ALTER TABLE EventTranslations ADD COLUMN IF NOT EXISTS Tickets Varchar NOT NULL DEFAULT '';
ALTER TABLE EventTranslations ADD COLUMN IF NOT EXISTS URLNames Varchar NOT NULL DEFAULT '';
ALTER TABLE EventTranslations ENABLE ROW LEVEL SECURITY;
