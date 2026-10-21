package firebase_client

import "context"

// User is the authenticated Firebase user extracted from a verified ID token.
type User struct {
	UID   string
	Email string
}

// Client verifies the ID tokens that the frontend gets when a user signs in with Firebase.
type Client interface {
	VerifyUser(ctx context.Context, idToken string) (User, error)
}
