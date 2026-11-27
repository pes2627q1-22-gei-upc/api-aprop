package s3_client

import (
	"context"
	"io"
)

// EventImageUploader uploads event posters.
type EventImageUploader interface {
	UploadEventImage(ctx context.Context, eventID int, filename string, contentType string, reader io.Reader) (string, error)
}

// AvatarRef is a profile picture as stored in Users.ProfilePicture and the user it belongs to.
type AvatarRef struct {
	UserID string
	Stored string
}

// AvatarSigner turns stored profile pictures into URLs that clients can load (#247).
type AvatarSigner interface {
	// AvatarURLs returns, for each avatar, the URL to show: empty and external (http/https) pictures are returned
	// as they are, avatars of the private bucket get a signed URL that expires, and anything else becomes empty.
	// On error the avatars that could not be signed are empty in the result, which always has len(avatars) items.
	AvatarURLs(ctx context.Context, avatars []AvatarRef) ([]string, error)
}

// AvatarStorage stores the profile pictures in a private bucket (#247).
type AvatarStorage interface {
	AvatarSigner
	// UploadAvatar uploads the avatar of userID and returns the value to store in Users.ProfilePicture (the
	// object path inside the bucket, not a URL).
	UploadAvatar(ctx context.Context, userID string, filename string, contentType string, reader io.Reader) (string, error)
	// DeleteAvatar deletes the avatar of userID stored as stored. Values that are not an avatar of that user
	// uploaded by this client (empty, external URLs, other users) are ignored.
	DeleteAvatar(ctx context.Context, userID string, stored string) error
}

// Client handles storage uploads for media assets like user avatars and event images.
type Client interface {
	EventImageUploader
	AvatarStorage
}
