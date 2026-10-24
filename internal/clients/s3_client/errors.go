package s3_client

import "errors"

var (
	ErrInvalidContentType = errors.New("invalid content type: only JPEG, PNG and WebP images are allowed")
	ErrFileTooLarge       = errors.New("file too large: maximum allowed size is 5MB")
	ErrUploadFailed       = errors.New("failed to upload avatar to storage")
)
