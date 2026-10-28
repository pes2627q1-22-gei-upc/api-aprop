package firebase_client

import (
	"context"
	"fmt"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
	"google.golang.org/api/option"
)

type FirebaseClient struct {
	authClient *auth.Client
}

// NewFirebaseClient creates a client authenticated with the service account JSON file at credentialsFile.
func NewFirebaseClient(ctx context.Context, credentialsFile string) (*FirebaseClient, error) {
	app, err := firebase.NewApp(ctx, nil, option.WithAuthCredentialsFile(option.ServiceAccount, credentialsFile))
	if err != nil {
		return nil, fmt.Errorf("init firebase app: %w", err)
	}

	authClient, err := app.Auth(ctx)
	if err != nil {
		return nil, fmt.Errorf("init firebase auth: %w", err)
	}

	return &FirebaseClient{authClient: authClient}, nil
}

// VerifyUser checks that the ID token is valid (signature, expiration and project) and that its user
// still exists in Firebase, is not disabled and has not revoked the token.
func (c *FirebaseClient) VerifyUser(ctx context.Context, idToken string) (User, error) {
	// VerifyIDTokenAndCheckRevoked also fetches the user from Firebase, so deleted or disabled users fail here
	token, err := c.authClient.VerifyIDTokenAndCheckRevoked(ctx, idToken)
	if err != nil {
		return User{}, mapAuthError(err)
	}

	email, _ := token.Claims["email"].(string)
	return User{UID: token.UID, Email: email}, nil
}

// mapAuthError converts Firebase SDK errors into the errors of this package.
func mapAuthError(err error) error {
	switch {
	case auth.IsUserNotFound(err):
		return ErrUserNotFound
	case auth.IsUserDisabled(err):
		return ErrUserDisabled
	case auth.IsIDTokenRevoked(err):
		return ErrTokenRevoked
	case auth.IsIDTokenInvalid(err), auth.IsIDTokenExpired(err):
		return ErrInvalidToken
	default:
		return fmt.Errorf("verify id token: %w", err)
	}
}
