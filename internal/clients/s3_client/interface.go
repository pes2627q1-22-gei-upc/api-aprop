package s3_client

import (
	"context"
	"io"
)

// Client handles storage uploads for media assets like user avatars and event images.
type Client interface {
	UploadAvatar(ctx context.Context, userID string, filename string, contentType string, reader io.Reader) (string, error)
	UploadEventImage(ctx context.Context, eventID int, filename string, contentType string, reader io.Reader) (string, error)
	// DeleteAvatar deletes the avatar of userID stored at url. URLs that are not an avatar of that user
	// uploaded by this client (empty, external, other users) are ignored.
	DeleteAvatar(ctx context.Context, userID string, url string) error
}
