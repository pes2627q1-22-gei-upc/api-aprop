package s3_client

import "errors"

var (
	ErrInvalidContentType = errors.New("invalid content type, must be image/jpeg, image/png, or image/webp")
	ErrUploadFailed       = errors.New("failed to upload image to storage")
	ErrFileTooLarge       = errors.New("file is too large (maximum 5MB)")
	ErrDeleteFailed       = errors.New("failed to delete file from storage")
)
