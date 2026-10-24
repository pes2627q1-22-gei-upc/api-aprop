package s3_client

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"
)

type S3Config struct {
	Bucket    string
	Region    string
	Endpoint  string // Optional, e.g. https://<project>.supabase.co/storage/v1/s3 or custom endpoint
	PublicURL string // Optional base URL to construct the public avatar URL
}

type S3Client struct {
	httpClient *http.Client
	config     S3Config
}

var _ Client = (*S3Client)(nil)

func NewS3Client(httpClient *http.Client, config S3Config) *S3Client {
	if config.Bucket == "" {
		config.Bucket = "avatars"
	}
	if config.Region == "" {
		config.Region = "eu-west-1"
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	return &S3Client{
		httpClient: httpClient,
		config:     config,
	}
}

func (c *S3Client) UploadAvatar(ctx context.Context, userID string, filename string, contentType string, reader io.Reader) (string, error) {
	// Validate MIME type
	switch contentType {
	case "image/jpeg", "image/png", "image/webp":
	default:
		return "", ErrInvalidContentType
	}

	ext := filepath.Ext(filename)
	if ext == "" {
		switch contentType {
		case "image/jpeg":
			ext = ".jpg"
		case "image/png":
			ext = ".png"
		case "image/webp":
			ext = ".webp"
		}
	}

	key := fmt.Sprintf("avatars/%s/avatar-%d%s", userID, time.Now().Unix(), ext)

	// If a custom HTTP endpoint is configured, perform HTTP PUT to store the asset
	if c.config.Endpoint != "" {
		uploadURL := fmt.Sprintf("%s/%s/%s", strings.TrimRight(c.config.Endpoint, "/"), c.config.Bucket, key)
		req, err := http.NewRequestWithContext(ctx, http.MethodPut, uploadURL, reader)
		if err != nil {
			return "", fmt.Errorf("create s3 upload request: %w", err)
		}
		req.Header.Set("Content-Type", contentType)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return "", fmt.Errorf("%w: %v", ErrUploadFailed, err)
		}
		defer resp.Body.Close()

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return "", fmt.Errorf("%w: storage responded with status %d", ErrUploadFailed, resp.StatusCode)
		}
	}

	// Determine public URL
	if c.config.PublicURL != "" {
		return fmt.Sprintf("%s/%s", strings.TrimRight(c.config.PublicURL, "/"), key), nil
	}
	if c.config.Endpoint != "" {
		return fmt.Sprintf("%s/%s/%s", strings.TrimRight(c.config.Endpoint, "/"), c.config.Bucket, key), nil
	}

	// Default standard AWS S3 public URL format
	return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", c.config.Bucket, c.config.Region, key), nil
}
