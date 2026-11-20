package s3_client

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

const (
	testUser   = "3f2504e0-4f89-11d3-9a0c-0305e82c3301"
	otherUser  = "9a2504e0-4f89-11d3-9a0c-0305e82c3302"
	testSecret = "sb_secret_test"
)

// fakeStorage records the requests to the Supabase Storage API.
type fakeStorage struct {
	mu       sync.Mutex
	requests []recordedRequest
	status   int
	// signResponse builds the body of POST /object/sign/{bucket} from the requested paths
	signResponse func(paths []string) any
}

type recordedRequest struct {
	method, path, apikey, authorization, contentType, upsert string
	body                                                     []byte
}

func newFakeStorage(t *testing.T) (*fakeStorage, *S3Client) {
	t.Helper()
	f := &fakeStorage{status: http.StatusOK}
	server := httptest.NewServer(http.HandlerFunc(f.serve))
	t.Cleanup(server.Close)
	c := NewS3Client(server.Client(), S3Config{SupabaseURL: server.URL + "/", SecretKey: testSecret})
	c.now = func() time.Time { return time.UnixMilli(1700000000123) }
	return f, c
}

func (f *fakeStorage) serve(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	f.mu.Lock()
	f.requests = append(f.requests, recordedRequest{
		method: r.Method, path: r.URL.Path, apikey: r.Header.Get("apikey"), authorization: r.Header.Get("Authorization"),
		contentType: r.Header.Get("Content-Type"), upsert: r.Header.Get("x-upsert"), body: body,
	})
	status, signResponse := f.status, f.signResponse
	f.mu.Unlock()

	if status != http.StatusOK {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(`{"error":"boom"}`))
		return
	}
	if strings.HasPrefix(r.URL.Path, "/storage/v1/object/sign/") {
		var req struct {
			ExpiresIn int      `json:"expiresIn"`
			Paths     []string `json:"paths"`
		}
		_ = json.Unmarshal(body, &req)
		var resp any
		if signResponse != nil {
			resp = signResponse(req.Paths)
		} else {
			items := []map[string]any{}
			for _, p := range req.Paths {
				items = append(items, map[string]any{"path": p, "error": nil, "signedURL": "/object/sign/avatars/" + p + "?token=t"})
			}
			resp = items
		}
		_ = json.NewEncoder(w).Encode(resp)
		return
	}
	_, _ = w.Write([]byte(`{}`))
}

func (f *fakeStorage) all() []recordedRequest {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]recordedRequest(nil), f.requests...)
}

func TestUploadAvatar(t *testing.T) {
	f, c := newFakeStorage(t)

	path, err := c.UploadAvatar(context.Background(), strings.ToUpper(testUser), "photo.weird ext", "image/jpeg", strings.NewReader("image-data"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if want := testUser + "/avatar-1700000000123.jpg"; path != want {
		t.Errorf("path = %q, want %q (extension from the content type, lowercase id)", path, want)
	}
	reqs := f.all()
	if len(reqs) != 1 {
		t.Fatalf("requests = %d, want 1", len(reqs))
	}
	r := reqs[0]
	if r.method != http.MethodPost || r.path != "/storage/v1/object/avatars/"+path {
		t.Errorf("request = %s %s", r.method, r.path)
	}
	if r.apikey != testSecret || r.authorization != "" {
		t.Errorf("apikey = %q, authorization = %q; want the secret key only in apikey", r.apikey, r.authorization)
	}
	if r.contentType != "image/jpeg" || r.upsert != "false" || string(r.body) != "image-data" {
		t.Errorf("content-type = %q, x-upsert = %q, body = %q", r.contentType, r.upsert, r.body)
	}
}

func TestUploadAvatar_Errors(t *testing.T) {
	t.Run("invalid content type", func(t *testing.T) {
		f, c := newFakeStorage(t)
		if _, err := c.UploadAvatar(context.Background(), testUser, "a.gif", "image/gif", strings.NewReader("x")); !errors.Is(err, ErrInvalidContentType) {
			t.Fatalf("err = %v, want ErrInvalidContentType", err)
		}
		if len(f.all()) != 0 {
			t.Error("request sent for an invalid content type")
		}
	})
	t.Run("not configured", func(t *testing.T) {
		c := NewS3Client(nil, S3Config{})
		if _, err := c.UploadAvatar(context.Background(), testUser, "a.png", "image/png", strings.NewReader("x")); !errors.Is(err, ErrStorageNotConfigured) {
			t.Fatalf("err = %v, want ErrStorageNotConfigured", err)
		}
	})
	t.Run("invalid user id", func(t *testing.T) {
		_, c := newFakeStorage(t)
		if _, err := c.UploadAvatar(context.Background(), "../events", "a.png", "image/png", strings.NewReader("x")); !errors.Is(err, ErrUploadFailed) {
			t.Fatalf("err = %v, want ErrUploadFailed", err)
		}
	})
	t.Run("storage error", func(t *testing.T) {
		f, c := newFakeStorage(t)
		f.status = http.StatusBadRequest
		if _, err := c.UploadAvatar(context.Background(), testUser, "a.png", "image/png", strings.NewReader("x")); !errors.Is(err, ErrUploadFailed) {
			t.Fatalf("err = %v, want ErrUploadFailed", err)
		}
	})
}

func TestStorageRequest_LegacyJWTKeyAlsoGoesInAuthorization(t *testing.T) {
	c := NewS3Client(nil, S3Config{SupabaseURL: "https://x.supabase.co", SecretKey: "eyJhbGciOiJIUzI1NiJ9.e30.sig"})
	req, err := c.storageRequest(context.Background(), http.MethodGet, "/bucket", nil)
	if err != nil {
		t.Fatal(err)
	}
	if req.Header.Get("Authorization") != "Bearer eyJhbGciOiJIUzI1NiJ9.e30.sig" || req.Header.Get("apikey") == "" {
		t.Errorf("headers = %v", req.Header)
	}
}

func TestDeleteAvatar(t *testing.T) {
	stored := testUser + "/avatar-1.png"
	tests := []struct {
		name     string
		userID   string
		stored   string
		status   int
		wantSent bool
		wantErr  error
	}{
		{name: "own avatar", userID: testUser, stored: stored, status: http.StatusOK, wantSent: true},
		{name: "storage error", userID: testUser, stored: stored, status: http.StatusInternalServerError, wantSent: true, wantErr: ErrDeleteFailed},
		{name: "empty", userID: testUser, stored: ""},
		{name: "external url", userID: testUser, stored: "https://lh3.googleusercontent.com/a/photo.jpg"},
		{name: "avatar of another user", userID: otherUser, stored: stored},
		{name: "path traversal", userID: testUser, stored: testUser + "/../" + otherUser + "/avatar-1.png"},
		{name: "old fake s3 url", userID: testUser, stored: "https://media.s3.eu-west-1.amazonaws.com/avatars/" + testUser + "/avatar-1.png"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, c := newFakeStorage(t)
			if tt.status != 0 {
				f.status = tt.status
			}

			err := c.DeleteAvatar(context.Background(), tt.userID, tt.stored)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}
			reqs := f.all()
			if !tt.wantSent {
				if len(reqs) != 0 {
					t.Errorf("sent %d requests, want none", len(reqs))
				}
				return
			}
			if len(reqs) != 1 || reqs[0].method != http.MethodDelete || reqs[0].path != "/storage/v1/object/avatars" {
				t.Fatalf("requests = %+v", reqs)
			}
			if string(reqs[0].body) != `{"prefixes":["`+stored+`"]}` || reqs[0].apikey != testSecret {
				t.Errorf("body = %s, apikey = %q", reqs[0].body, reqs[0].apikey)
			}
		})
	}
}

func TestAvatarURLs(t *testing.T) {
	f, c := newFakeStorage(t)
	own := testUser + "/avatar-1.png"
	other := otherUser + "/avatar-2.webp"

	urls, err := c.AvatarURLs(context.Background(), []AvatarRef{
		{UserID: testUser, Stored: own},
		{UserID: otherUser, Stored: "https://example.com/me.png"},
		{UserID: otherUser, Stored: ""},
		{UserID: otherUser, Stored: own}, // a path of another user is hidden
		{UserID: testUser, Stored: "avatars/" + own},
		{UserID: otherUser, Stored: other},
		{UserID: testUser, Stored: own}, // duplicates are signed once
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	base := c.config.SupabaseURL + "/storage/v1/object/sign/avatars/"
	want := []string{base + own + "?token=t", "https://example.com/me.png", "", "", "", base + other + "?token=t", base + own + "?token=t"}
	for i := range want {
		if urls[i] != want[i] {
			t.Errorf("urls[%d] = %q, want %q", i, urls[i], want[i])
		}
	}

	reqs := f.all()
	if len(reqs) != 1 || reqs[0].method != http.MethodPost || reqs[0].path != "/storage/v1/object/sign/avatars" {
		t.Fatalf("requests = %+v, want one batch sign request", reqs)
	}
	var body struct {
		ExpiresIn int      `json:"expiresIn"`
		Paths     []string `json:"paths"`
	}
	if err := json.Unmarshal(reqs[0].body, &body); err != nil {
		t.Fatal(err)
	}
	if body.ExpiresIn != 3600 || len(body.Paths) != 2 || body.Paths[0] != own || body.Paths[1] != other {
		t.Errorf("sign body = %+v", body)
	}
}

func TestAvatarURLs_Cache(t *testing.T) {
	f, c := newFakeStorage(t)
	now := time.Unix(1700000000, 0)
	c.now = func() time.Time { return now }
	refs := []AvatarRef{{UserID: testUser, Stored: testUser + "/avatar-1.png"}}

	first, _ := c.AvatarURLs(context.Background(), refs)
	now = now.Add(29 * time.Minute) // still valid for more than half of the TTL
	second, _ := c.AvatarURLs(context.Background(), refs)
	if len(f.all()) != 1 || first[0] != second[0] {
		t.Fatalf("requests = %d, urls %q / %q; want the cached URL", len(f.all()), first[0], second[0])
	}

	now = now.Add(2 * time.Minute) // less than half of the TTL left
	if _, err := c.AvatarURLs(context.Background(), refs); err != nil {
		t.Fatal(err)
	}
	if len(f.all()) != 2 {
		t.Errorf("requests = %d, want the URL signed again", len(f.all()))
	}

	if err := c.DeleteAvatar(context.Background(), testUser, refs[0].Stored); err != nil {
		t.Fatal(err)
	}
	if _, ok := c.signedURLs[refs[0].Stored]; ok {
		t.Error("deleted avatar still cached")
	}
}

func TestAvatarURLs_Errors(t *testing.T) {
	refs := []AvatarRef{{UserID: testUser, Stored: testUser + "/avatar-1.png"}, {UserID: testUser, Stored: "https://example.com/me.png"}}

	t.Run("storage error keeps external urls", func(t *testing.T) {
		f, c := newFakeStorage(t)
		f.status = http.StatusInternalServerError
		urls, err := c.AvatarURLs(context.Background(), refs)
		if !errors.Is(err, ErrSignFailed) {
			t.Fatalf("err = %v, want ErrSignFailed", err)
		}
		if urls[0] != "" || urls[1] != "https://example.com/me.png" {
			t.Errorf("urls = %q", urls)
		}
	})
	t.Run("not configured", func(t *testing.T) {
		urls, err := NewS3Client(nil, S3Config{}).AvatarURLs(context.Background(), refs)
		if !errors.Is(err, ErrStorageNotConfigured) || urls[0] != "" || urls[1] != "https://example.com/me.png" {
			t.Errorf("urls = %q, err = %v", urls, err)
		}
	})
	t.Run("missing object is hidden", func(t *testing.T) {
		f, c := newFakeStorage(t)
		f.signResponse = func(paths []string) any {
			return []map[string]any{{"path": paths[0], "error": "Either the object does not exist or you do not have access to it", "signedURL": nil}}
		}
		urls, err := c.AvatarURLs(context.Background(), refs[:1])
		if err != nil || urls[0] != "" {
			t.Errorf("urls = %q, err = %v; want an empty url", urls, err)
		}
	})
}

func TestResolveAvatars(t *testing.T) {
	stored, external := testUser+"/avatar-1.png", "https://example.com/me.png"
	a, b, empty := stored, external, ""
	fields := []AvatarField{{UserID: testUser, Picture: &a}, {UserID: testUser, Picture: &b}, {UserID: testUser, Picture: &empty}}

	if err := ResolveAvatars(context.Background(), nil, fields); err != nil {
		t.Fatal(err)
	}
	if a != "" || b != external || empty != "" {
		t.Errorf("without signer: %q %q %q", a, b, empty)
	}

	_, c := newFakeStorage(t)
	a = stored
	if err := ResolveAvatars(context.Background(), c, fields); err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(a, "/storage/v1/object/sign/avatars/"+stored+"?token=t") || b != external {
		t.Errorf("with signer: %q %q", a, b)
	}
}
