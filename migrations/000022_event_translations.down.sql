DROP TABLE IF EXISTS EventTranslations;
ALTER TABLE Settings DROP CONSTRAINT IF EXISTS settings_language_check;
ALTER TABLE Settings DROP COLUMN IF EXISTS Language;
