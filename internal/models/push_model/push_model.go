// Package push_model contains the push notifications domain: the devices of the users and the notifications of the
// inbox pending to be pushed (Taiga #275).
package push_model

// Platforms accepted for a device.
const (
	PlatformAndroid = "android"
	PlatformIOS     = "ios"
	PlatformWeb     = "web"
)

// MaxTokenLength bounds the FCM registration token accepted from a client (current tokens are ~160 characters).
const MaxTokenLength = 4096

// Device is a device of a user that receives push notifications, identified by its FCM registration token.
type Device struct {
	Token    string `json:"token"`
	Platform string `json:"platform"`
}

// PendingPush is a notification of the inbox (Notifications) that has not been pushed yet.
type PendingPush struct {
	ID          int
	UserID      string
	Type        string // name of the NotificationType: Social, Recomanacio, Recordatori, Cancelacio
	Text        string
	EventID     *int
	FriendID    *string
	PushEnabled bool // Settings.PushEnabled of the user
}

// ValidPlatform reports whether platform is one of the accepted platforms.
func ValidPlatform(platform string) bool {
	switch platform {
	case PlatformAndroid, PlatformIOS, PlatformWeb:
		return true
	}
	return false
}
