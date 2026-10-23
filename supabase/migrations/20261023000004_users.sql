-- Every new user starts in this state (Users.AccountState is NOT NULL)
INSERT INTO AccountStates (Name) VALUES ('Active') ON CONFLICT (Name) DO NOTHING;

-- Users reference their settings by ID, so Settings no longer has a UserID key
ALTER TABLE Settings DROP CONSTRAINT settings_pkey;
ALTER TABLE Settings DROP COLUMN UserID;
ALTER TABLE Settings ADD COLUMN ID Integer GENERATED ALWAYS AS IDENTITY PRIMARY KEY;

-- Users keep the profile information. FirebaseUID links each row with its Firebase Auth account.
CREATE TABLE Users (
    ID Integer GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    FirebaseUID Varchar UNIQUE NOT NULL,
    Username Varchar UNIQUE NOT NULL,
    Email Varchar,
    ProfilePicture Varchar,
    Biography Varchar,
    DateRegistered Timestamp NOT NULL DEFAULT now(),
    AccountState Integer NOT NULL REFERENCES AccountStates(ID),
    Experience Integer NOT NULL DEFAULT 0,
    Level Integer REFERENCES Levels(ID),
    Rating Real,
    Settings Integer UNIQUE NOT NULL REFERENCES Settings(ID)
);
