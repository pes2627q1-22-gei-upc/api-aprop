package s3_client_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"api-aprop/internal/clients/s3_client"
)

func TestUploadAvatar_InvalidContentType(t *testing.T) {
	c := s3_client.NewS3Client(nil, s3_client.S3Config{})
	_, err := c.UploadAvatar(context.Background(), "user-123", "test.gif", "image/gif", strings.NewReader("fake"))
	if err != s3_client.ErrInvalidContentType {
		t.Fatalf("expected ErrInvalidContentType, got %v", err)
	}
}

func TestUploadAvatar_DefaultURL(t *testing.T) {
	c := s3_client.NewS3Client(nil, s3_client.S3Config{
		Bucket: "my-bucket",
		Region: "eu-west-1",
	})
	url, err := c.UploadAvatar(context.Background(), "user-123", "avatar.png", "image/png", strings.NewReader("fake-png"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasPrefix(url, "https://my-bucket.s3.eu-west-1.amazonaws.com/avatars/user-123/avatar-") {
		t.Errorf("unexpected url format: %s", url)
	}
	if !strings.HasSuffix(url, ".png") {
		t.Errorf("expected .png suffix, got %s", url)
	}
}

func TestUploadAvatar_WithEndpoint(t *testing.T) {
	var receivedContentType string
	var receivedBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedContentType = r.Header.Get("Content-Type")
		receivedBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c := s3_client.NewS3Client(server.Client(), s3_client.S3Config{
		Bucket:   "avatars",
		Endpoint: server.URL,
	})

	data := []byte("image-data")
	url, err := c.UploadAvatar(context.Background(), "user-456", "photo.jpg", "image/jpeg", bytes.NewReader(data))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if receivedContentType != "image/jpeg" {
		t.Errorf("expected image/jpeg, got %s", receivedContentType)
	}
	if string(receivedBody) != "image-data" {
		t.Errorf("expected image-data, got %s", string(receivedBody))
	}
	if !strings.HasPrefix(url, server.URL+"/avatars/avatars/user-456/avatar-") {
		t.Errorf("unexpected url: %s", url)
	}
}
