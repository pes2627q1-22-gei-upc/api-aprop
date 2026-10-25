package user_model

import (
	"time"
	"github.com/jackc/pgx/v5/pgtype"
)

// User is the profile stored in the database. FirebaseUID links it with its Firebase Auth account.
type User struct {
	ID             pgtype.UUID 	`json:"id"`
	FirebaseUID    string    	`json:"firebase-uid"`
	Username       string    	`json:"username"`
	Email          string    	`json:"email"`
	ProfilePicture string    	`json:"profile-picture"`
	Biography      string    	`json:"biography"`
	DateRegistered time.Time 	`json:"date-registered"`
	AccountState   int       	`json:"account-state"`
	Experience     int       	`json:"experience"`
	Level          *int      	`json:"level"`  // nil until the user reaches the first level
	Rating         *float32  	`json:"rating"` // nil until the user is rated
	Settings       int       	`json:"settings"`
}

// SyncResult is the result of syncing a Firebase user with the database.
type SyncResult struct {
	User    User `json:"user"`
	Created bool `json:"created"` // true when the user signed in for the first time
}
