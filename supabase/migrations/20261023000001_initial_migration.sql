CREATE TABLE Places (
    ID Integer GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    Name Varchar NOT NULL,
    Address Varchar NOT NULL,
    Municipality Varchar,
    Region Varchar,
    Latitude Real,
    Longitude Real,
    PostalCode Varchar,
    UNIQUE (Name, Address),
    CHECK (Latitude <= 90 AND Latitude >= -90),
    CHECK (Longitude <= 180 AND Longitude >= -180)
);

CREATE TABLE Events (
    ID Integer GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    Title Varchar NOT NULL,
    Subtitle Varchar,
    Description Varchar NOT NULL,
    Schedule Varchar,
    URL Varchar,
    Tickets Varchar,
    TicketsURL Varchar,
    Free Boolean NOT NULL,
    Permanent Boolean NOT NULL,
    Online Boolean NOT NULL,
    RelatedURLs Varchar,
    URLNames Varchar,
    Images Varchar,
    Videos Varchar,
    Email Varchar,
    PlaceID Integer REFERENCES Places(ID),
    Phone Varchar,
    ActivityURL Varchar,
    StartDate Date NOT NULL,
    EndDate Date,
    CHECK (EndDate >= StartDate)
);

CREATE TABLE OfficialEvents (
    ID Integer REFERENCES Events(ID),
    Code Bigint,
    PRIMARY KEY (ID)
);

CREATE TABLE UnofficialEvents (
    ID Integer REFERENCES Events(ID),
    Creator UUID,
    DateCreated Timestamp,
    PRIMARY KEY (ID)
);

CREATE TABLE AccountStates (
    ID Integer GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    Name Varchar UNIQUE NOT NULL
);

CREATE TABLE Levels (
    ID Integer GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    Name Varchar UNIQUE NOT NULL,
    ExperienceRequired Integer UNIQUE NOT NULL
);

CREATE TABLE UserLevels (
    UserID UUID,
    Level Integer REFERENCES Levels(ID),
    PRIMARY KEY (UserID)
);

CREATE TABLE UserState (
    UserID UUID,
    State Integer REFERENCES AccountStates(ID),
    PRIMARY KEY (UserID)
);

CREATE TABLE Settings (
    UserID UUID,
    PRIMARY KEY (UserID)
    -- Complete later
);

CREATE TABLE PreferenceTests (
    UserID UUID,
    DateTaken Timestamp NOT NULL,
    PRIMARY KEY (UserID, DateTaken)
);

CREATE TABLE PreferenceTestChoices (
    TestUser UUID,
    DateTaken Timestamp,
    Event Integer REFERENCES Events(ID),
    Likes Boolean NOT NULL,
    PRIMARY KEY (TestUser, DateTaken, Event),
    FOREIGN KEY (TestUser, DateTaken) REFERENCES PreferenceTests(UserID, DateTaken)
);

CREATE TABLE Friendships (
    User1 UUID,
    User2 UUID,
    PRIMARY KEY (User1, User2),
    CHECK (User1 <> User2)
);

CREATE TABLE Blocked (
    BlockingUser UUID,
    BlockedUser UUID,
    PRIMARY KEY (BlockingUser, BlockedUser),
    CHECK (BlockingUser <> BlockedUser)
);

CREATE TYPE FriendRequestState AS ENUM ('Pending', 'Accepted', 'Rejected');

CREATE TABLE FriendRequests (
    Sender UUID,
    Receiver UUID,
    DateSent Timestamp NOT NULL,
    State FriendRequestState NOT NULL DEFAULT 'Pending',
    DateResolved Timestamp,
    PRIMARY KEY (Sender, Receiver, DateSent),
    CHECK (Sender <> Receiver)
);

CREATE TABLE Categories (
    ID Integer GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    Name Varchar UNIQUE NOT NULL
);

CREATE TABLE EventCategories (
    Event Integer REFERENCES Events(ID),
    Category Integer REFERENCES Categories(ID),
    PRIMARY KEY (Event, Category)
);

CREATE TABLE UserPreferences (
    UserID UUID,
    Category Integer REFERENCES Categories(ID),
    Weight Real NOT NULL DEFAULT 1,
    PRIMARY KEY (UserID, Category)
);

CREATE TABLE SignUp (
    UserID UUID,
    Event Integer REFERENCES Events(ID),
    SignUpDate Timestamp NOT NULL,
    PRIMARY KEY (UserID, Event, SignUpDate)
);

CREATE TABLE Messages (
    UserID UUID,
    Event Integer REFERENCES Events(ID),
    DateSent Timestamp NOT NULL,
    Message Varchar NOT NULL,
    PRIMARY KEY (UserID, Event, DateSent)
);

CREATE TABLE EventRatings (
    UserID UUID,
    Event Integer REFERENCES Events(ID),
    RatingDate Timestamp NOT NULL,
    Rating Integer NOT NULL,
    PRIMARY KEY (UserID, Event),
    CHECK (Rating >= 0 AND Rating <= 5)
);

CREATE TABLE UserRatings (
    Rater UUID,
    Rated UUID,
    Event Integer REFERENCES Events(ID),
    RatingDate Timestamp NOT NULL,
    PRIMARY KEY (Rater, Rated, Event),
    CHECK (Rater <> Rated)
);

CREATE TABLE NotificationTypes (
    ID Integer GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    Name Varchar UNIQUE NOT NULL
);

CREATE TABLE Notifications (
    UserID UUID,
    NotificationDate Timestamp NOT NULL,
    Friend UUID,
    RequestDate Timestamp,
    Event Integer REFERENCES Events(ID),
    Type Integer REFERENCES NotificationTypes(ID),
    PRIMARY KEY (UserID, NotificationDate),
    CHECK (UserID <> Friend),
    FOREIGN KEY (Friend, UserID, RequestDate) REFERENCES FriendRequests(Sender, Receiver, DateSent)
);