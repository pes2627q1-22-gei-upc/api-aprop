package s3_client

import (
	"context"
	"io"
)

// Client handles storage uploads for media assets like user avatars.
type Client interface {
	UploadAvatar(ctx context.Context, userID string, filename string, contentType string, reader io.Reader) (string, error)
}
