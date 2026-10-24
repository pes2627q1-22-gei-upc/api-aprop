package s3_client

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
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
