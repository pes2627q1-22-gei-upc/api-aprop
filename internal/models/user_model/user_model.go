package user_model

import (
	"github.com/jackc/pgx/v5/pgtype"
	"time"
)

// User is the profile stored in the database. FirebaseUID links it with its Firebase Auth account.
type User struct {
	ID             pgtype.UUID `json:"id"`
	FirebaseUID    string      `json:"firebase-uid"`
	Username       string      `json:"username"`
	Name           string      `json:"name,omitempty"`
	Email          string      `json:"email"`
	ProfilePicture string      `json:"profile-picture"`
	Biography      string      `json:"biography"`
	DateRegistered time.Time   `json:"date-registered"`
	AccountState   int         `json:"account-state"`
	Experience     int         `json:"experience"`
	Level          *int        `json:"level"`  // nil until the user reaches the first level
	Rating         *float32    `json:"rating"` // nil until the user is rated
	Settings       int         `json:"settings"`
}

// UserProfile is the full or public profile representation with derived counters and privacy settings.
type UserProfile struct {
	ID                  pgtype.UUID `json:"id"`
	FirebaseUID         string      `json:"firebase-uid,omitempty"`
	Username            string      `json:"username"`
	Name                string      `json:"name"`
	Email               string      `json:"email,omitempty"`
	ProfilePicture      string      `json:"profile-picture"`
	Biography           string      `json:"biography"`
	DateRegistered      time.Time   `json:"date-registered"`
	AccountState        int         `json:"account-state"`
	Experience          int         `json:"experience"`
	Level               *int        `json:"level"`
	Rating              *float32    `json:"rating"`
	Reliability         *float32    `json:"reliability"`
	FriendsCount        int         `json:"friends-count"`
	PlansCount          int         `json:"plans-count"`
	CompletedPlansCount int         `json:"completed-plans-count"`
	ShowPlans           bool        `json:"show-plans"`
	IsFriend            *bool       `json:"is-friend,omitempty"`
	PlansVisible        bool        `json:"plans-visible"`
	NotificationBefore  string      `json:"notification-before"`
}

// SyncResult is the result of syncing a Firebase user with the database.
type SyncResult struct {
	User    User `json:"user"`
	Created bool `json:"created"` // true when the user signed in for the first time
}

type FriendRequest struct {
	Sender       pgtype.UUID `json:"sender-id"`
	Receiver     pgtype.UUID `json:"receiver-id"`
	DateSent     time.Time   `json:"date-sent"`
	State        string      `json:"state"`
	DateResolved time.Time   `json:"date-resolved"`
}
