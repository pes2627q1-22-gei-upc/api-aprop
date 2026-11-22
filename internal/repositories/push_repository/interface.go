package push_repository

import (
	"context"
	"time"

	"api-aprop/internal/models/push_model"
)

// SendFunc pushes a batch of claimed notifications and returns the IDs of the ones that are resolved (sent, or that
// must not be retried). The rest stay pending for a later dispatch.
type SendFunc func(ctx context.Context, batch []push_model.PendingPush) (resolvedIDs []int, err error)

// Repository stores the devices of the users (DeviceTokens) and the outbox of the push notifications
// (Notifications.PushedAt).
type Repository interface {
	// UpsertDevice registers the device for the user; an existing token moves to this user.
	UpsertDevice(ctx context.Context, userID string, device push_model.Device) error
	// DeleteDevice removes the token of the user and reports whether it existed.
	DeleteDevice(ctx context.Context, userID, token string) (bool, error)
	// DeleteDeviceTokens removes tokens that FCM no longer accepts, whoever their user is.
	DeleteDeviceTokens(ctx context.Context, tokens []string) error
	// DeviceTokens returns the tokens of each user, the most recently registered first.
	DeviceTokens(ctx context.Context, userIDs []string) (map[string][]string, error)
	// ExpirePending marks as pushed, without sending them, the pending notifications older than maxAge.
	ExpirePending(ctx context.Context, maxAge time.Duration) (int64, error)
	// ProcessPending claims up to limit pending notifications in a transaction, calls send with them and marks as
	// pushed the IDs it returns. It returns the number of claimed notifications (0 when the outbox is empty).
	// When send fails, nothing is marked and the error is returned.
	ProcessPending(ctx context.Context, limit int, send SendFunc) (int, error)
}
