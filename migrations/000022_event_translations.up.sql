-- Language of the user interface and of the event texts (US #28). TODO(DP-043)
ALTER TABLE Settings ADD COLUMN IF NOT EXISTS Language Varchar NOT NULL DEFAULT 'ca';
ALTER TABLE Settings DROP CONSTRAINT IF EXISTS settings_language_check;
ALTER TABLE Settings ADD CONSTRAINT settings_language_check CHECK (Language IN ('ca', 'es', 'en'));

-- Cache of the machine translations of the title and description of an event. The originals are in Catalan.
-- SourceHash is the SHA-256 (hex) of the Catalan title and description that were translated: when the event changes,
-- the hash no longer matches and the row is translated again.
CREATE TABLE IF NOT EXISTS EventTranslations (
    Event Integer NOT NULL REFERENCES Events(ID) ON DELETE CASCADE,
    Lang Varchar NOT NULL CHECK (Lang IN ('es', 'en')),
    Title Varchar NOT NULL,
    Description Varchar NOT NULL,
    SourceHash Varchar NOT NULL,
    UpdatedAt Timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (Event, Lang)
);
ALTER TABLE EventTranslations ENABLE ROW LEVEL SECURITY;
