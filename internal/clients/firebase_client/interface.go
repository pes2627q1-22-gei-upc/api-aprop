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

// PushMessage is a push notification: a visible title and body plus the data the app uses to open the right screen.
type PushMessage struct {
	Title string
	Body  string
	Data  map[string]string
}

// PushResult summarizes the delivery of a PushMessage to every device of a user.
type PushResult struct {
	SuccessCount int
	// InvalidTokens are the devices that FCM no longer accepts (app uninstalled, token expired or of another project).
	InvalidTokens []string
	// TransientFailures counts the devices that failed for a reason worth retrying (FCM unavailable, quota...).
	TransientFailures int
}

// PushSender sends push notifications through Firebase Cloud Messaging.
type PushSender interface {
	// Send pushes msg to the devices, identified by FCM registration tokens or Firebase Installation IDs. It only
	// returns an error when nothing could be sent (e.g. bad credentials); per device failures are in PushResult.
	Send(ctx context.Context, tokens []string, msg PushMessage) (PushResult, error)
}

// Client verifies the ID tokens that the frontend gets when a user signs in with Firebase.
type Client interface {
	VerifyUser(ctx context.Context, idToken string) (User, error)
}
