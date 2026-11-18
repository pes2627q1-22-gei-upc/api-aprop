package firebase_client

import "context"

// User is the authenticated Firebase user extracted from a verified ID token.
type User struct {
	UID   string
	Email string
}

// AccountDeleter deletes Firebase Auth accounts. It is apart from Client so the auth middleware and
// its fakes do not need it.
type AccountDeleter interface {
	// DeleteUser deletes the account with the given UID. An account that does not exist is not an error.
	DeleteUser(ctx context.Context, uid string) error
}

// PasswordResetSender makes Firebase Auth send the password reset email of an account.
type PasswordResetSender interface {
	// SendPasswordResetEmail asks Firebase to send the reset link to email, written in locale (e.g. "ca").
	// It returns ErrEmailNotFound, ErrInvalidEmail or ErrTooManyAttempts when Firebase reports them.
	SendPasswordResetEmail(ctx context.Context, email, locale string) error
}

// Client verifies the ID tokens that the frontend gets when a user signs in with Firebase.
type Client interface {
	VerifyUser(ctx context.Context, idToken string) (User, error)
}
