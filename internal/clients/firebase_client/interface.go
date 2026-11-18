package firebase_client

import "context"

// User is the authenticated Firebase user extracted from a verified ID token.
type User struct {
	UID   string
	Email string
}

// Client verifies the ID tokens that the frontend gets when a user signs in with Firebase.
// AccountDeleter deletes Firebase Auth accounts. It is apart from Client so the auth middleware and
// its fakes do not need it.
type AccountDeleter interface {
	// DeleteUser deletes the account with the given UID. An account that does not exist is not an error.
	DeleteUser(ctx context.Context, uid string) error
}

type Client interface {
	VerifyUser(ctx context.Context, idToken string) (User, error)
}
