package push_service

import "context"

// Service registers the devices of the users and pushes the notifications of their inbox (Taiga #275).
type Service interface {
	// RegisterDevice stores the FCM token (or FID) of a device of the user; registering it again is not an error.
	RegisterDevice(ctx context.Context, userID, token, platform string) error
	// UnregisterDevice forgets a device of the user, e.g. when signing out. A device that is not registered is not an error.
	UnregisterDevice(ctx context.Context, userID, token string) error
	// DispatchPending pushes the notifications of the outbox and returns how many were delivered to some device.
	DispatchPending(ctx context.Context) (int, error)
}
