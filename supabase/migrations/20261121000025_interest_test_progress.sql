-- Tracks whether the user has already been prompted with the onboarding interest test
-- so skipping or closing it never forces it again on subsequent logins.
-- Existing accounts default to true; newly registered accounts default to false.
-- TODO(DP-047)
ALTER TABLE Users ADD COLUMN IF NOT EXISTS InterestTestPrompted Boolean NOT NULL DEFAULT true;
ALTER TABLE Users ALTER COLUMN InterestTestPrompted SET DEFAULT false;

-- Stores partial ("Més tard") interest test answers without overwriting any previously
-- completed PreferenceTests / UserPreferences rows (R1).
CREATE TABLE IF NOT EXISTS PreferenceTestProgress (
    UserID UUID NOT NULL REFERENCES Users(ID) ON DELETE CASCADE,
    Event Integer NOT NULL REFERENCES Events(ID) ON DELETE CASCADE,
    Likes Boolean NOT NULL,
    UpdatedAt Timestamp NOT NULL DEFAULT now(),
    PRIMARY KEY (UserID, Event)
);

ALTER TABLE PreferenceTestProgress ENABLE ROW LEVEL SECURITY;
