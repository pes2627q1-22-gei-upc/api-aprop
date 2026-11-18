package s3_client

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type roundTripFunc func(req *http.Request) *http.Response

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req), nil
}

func TestUploadEventImage(t *testing.T) {
	tests := []struct {
		name        string
		eventID     int
		filename    string
		contentType string
		wantErr     error
		wantSub     string
	}{
		{
			name:        "valid jpeg image",
			eventID:     12,
			filename:    "concert.jpg",
			contentType: "image/jpeg",
			wantSub:     "events/12/poster-",
		},
		{
			name:        "valid png image without extension",
			eventID:     45,
			filename:    "blob",
			contentType: "image/png",
			wantSub:     "events/45/poster-",
		},
		{
			name:        "invalid mime type",
			eventID:     12,
			filename:    "bad.pdf",
			contentType: "application/pdf",
			wantErr:     ErrInvalidContentType,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewS3Client(nil, S3Config{
				Bucket: "media",
				Region: "eu-west-1",
			})

			url, err := client.UploadEventImage(context.Background(), tt.eventID, tt.filename, tt.contentType, strings.NewReader("fake-content"))
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr != nil {
				return
			}
			if !strings.Contains(url, tt.wantSub) {
				t.Errorf("url %q does not contain %q", url, tt.wantSub)
			}
		})
	}
}

func TestUploadWithEndpoint(t *testing.T) {
	var gotPath string
	var gotContentType string

	mockTransport := roundTripFunc(func(req *http.Request) *http.Response {
		gotPath = req.URL.Path
		gotContentType = req.Header.Get("Content-Type")
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewBufferString("ok")),
			Header:     make(http.Header),
		}
	})

	httpClient := &http.Client{Transport: mockTransport}
	client := NewS3Client(httpClient, S3Config{
		Bucket:   "test-bucket",
		Endpoint: "http://storage.local",
	})

	url, err := client.UploadEventImage(context.Background(), 99, "poster.png", "image/png", strings.NewReader("content"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.HasPrefix(url, "http://storage.local/test-bucket/events/99/poster-") {
		t.Errorf("unexpected public url: %q", url)
	}
	if !strings.HasPrefix(gotPath, "/test-bucket/events/99/poster-") {
		t.Errorf("unexpected request path: %q", gotPath)
	}
	if gotContentType != "image/png" {
		t.Errorf("Content-Type = %q, want image/png", gotContentType)
	}
}

func TestUploadAvatar_InvalidContentType(t *testing.T) {
	c := NewS3Client(nil, S3Config{})
	_, err := c.UploadAvatar(context.Background(), "user-123", "test.gif", "image/gif", strings.NewReader("fake"))
	if !errors.Is(err, ErrInvalidContentType) {
		t.Fatalf("expected ErrInvalidContentType, got %v", err)
	}
}

func TestUploadAvatar_DefaultURL(t *testing.T) {
	c := NewS3Client(nil, S3Config{
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

	c := NewS3Client(server.Client(), S3Config{
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

func TestDeleteAvatar(t *testing.T) {
	const userID = "3f2504e0-4f89-11d3-9a0c-0305e82c3301"
	tests := []struct {
		name     string
		config   S3Config
		url      string
		status   int
		wantPath string // empty: no request expected
		wantErr  error
	}{
		{
			name:     "endpoint url",
			config:   S3Config{Bucket: "media", Endpoint: "http://storage.local/s3/"},
			url:      "http://storage.local/s3/media/avatars/" + userID + "/avatar-1.png",
			status:   http.StatusNoContent,
			wantPath: "/s3/media/avatars/" + userID + "/avatar-1.png",
		},
		{
			name:     "public url",
			config:   S3Config{Bucket: "media", Endpoint: "http://storage.local", PublicURL: "https://cdn.aprop.cat/public"},
			url:      "https://cdn.aprop.cat/public/avatars/" + userID + "/avatar-1.png",
			status:   http.StatusOK,
			wantPath: "/media/avatars/" + userID + "/avatar-1.png",
		},
		{
			name:     "already deleted",
			config:   S3Config{Bucket: "media", Endpoint: "http://storage.local"},
			url:      "http://storage.local/media/avatars/" + userID + "/avatar-1.png",
			status:   http.StatusNotFound,
			wantPath: "/media/avatars/" + userID + "/avatar-1.png",
		},
		{
			name:     "storage error",
			config:   S3Config{Bucket: "media", Endpoint: "http://storage.local"},
			url:      "http://storage.local/media/avatars/" + userID + "/avatar-1.png",
			status:   http.StatusInternalServerError,
			wantPath: "/media/avatars/" + userID + "/avatar-1.png",
			wantErr:  ErrDeleteFailed,
		},
		{name: "empty url", config: S3Config{Bucket: "media", Endpoint: "http://storage.local"}, url: ""},
		{name: "external url", config: S3Config{Bucket: "media", Endpoint: "http://storage.local"}, url: "https://lh3.googleusercontent.com/a/photo.jpg"},
		{name: "avatar of another user", config: S3Config{Bucket: "media", Endpoint: "http://storage.local"}, url: "http://storage.local/media/avatars/other/avatar-1.png"},
		{name: "event image", config: S3Config{Bucket: "media", Endpoint: "http://storage.local"}, url: "http://storage.local/media/events/1/poster-1.png"},
		{name: "path traversal", config: S3Config{Bucket: "media", Endpoint: "http://storage.local"}, url: "http://storage.local/media/avatars/" + userID + "/../../events/1/poster.png"},
		{name: "no endpoint", config: S3Config{Bucket: "media"}, url: "https://media.s3.eu-west-1.amazonaws.com/avatars/" + userID + "/avatar-1.png"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotMethod, gotPath string
			httpClient := &http.Client{Transport: roundTripFunc(func(req *http.Request) *http.Response {
				gotMethod, gotPath = req.Method, req.URL.Path
				return &http.Response{StatusCode: tt.status, Body: io.NopCloser(strings.NewReader("")), Header: make(http.Header)}
			})}

			err := NewS3Client(httpClient, tt.config).DeleteAvatar(context.Background(), userID, tt.url)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}
			if gotPath != tt.wantPath {
				t.Errorf("request path = %q, want %q", gotPath, tt.wantPath)
			}
			if tt.wantPath != "" && gotMethod != http.MethodDelete {
				t.Errorf("method = %s, want DELETE", gotMethod)
			}
		})
	}
}
