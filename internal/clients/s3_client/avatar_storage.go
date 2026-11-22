package s3_client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// DefaultAvatarURLTTL is the lifetime of the signed avatar URLs.
// TODO(DP-045): proposed value, not agreed with the frontend.
const DefaultAvatarURLTTL = time.Hour

// maxCachedSignedURLs bounds the cache of signed URLs; expired entries are dropped when it is reached.
const maxCachedSignedURLs = 5000

// avatarPathRegex matches the object paths created by UploadAvatar: <user uuid>/avatar-<unix ms>.<ext>.
var avatarPathRegex = regexp.MustCompile(`^([0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})/avatar-[0-9]+\.(jpg|png|webp)$`)

type signedURL struct {
	url       string
	expiresAt time.Time
}

// UploadAvatar uploads the avatar to the private bucket with the Supabase Storage API and returns its object path
// (<user id>/avatar-<unix ms>.<ext>), the value to store in Users.ProfilePicture.
func (c *S3Client) UploadAvatar(ctx context.Context, userID string, _ string, contentType string, reader io.Reader) (string, error) {
	ext, ok := avatarExtension(contentType)
	if !ok {
		return "", ErrInvalidContentType
	}
	userID = strings.ToLower(userID)
	path := fmt.Sprintf("%s/avatar-%d%s", userID, c.now().UnixMilli(), ext)
	if !avatarPathRegex.MatchString(path) {
		return "", fmt.Errorf("%w: invalid user id %q", ErrUploadFailed, userID)
	}
	if !c.avatarsConfigured() {
		return "", ErrStorageNotConfigured
	}

	req, err := c.storageRequest(ctx, http.MethodPost, "/object/"+c.config.AvatarBucket+"/"+path, reader)
	if err != nil {
		return "", fmt.Errorf("create avatar upload request: %w", err)
	}
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("x-upsert", "false")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrUploadFailed, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("%w: storage responded with status %d: %s", ErrUploadFailed, resp.StatusCode, readSnippet(resp.Body))
	}
	return path, nil
}

// DeleteAvatar deletes the avatar of userID from the private bucket. It is idempotent: deleting a missing
// object succeeds. Values that are not an avatar path of that user are ignored.
func (c *S3Client) DeleteAvatar(ctx context.Context, userID string, stored string) error {
	if !isAvatarOf(stored, userID) || !c.avatarsConfigured() {
		return nil
	}

	// Deleting by prefixes answers 200 also when the object does not exist.
	body, err := json.Marshal(map[string][]string{"prefixes": {stored}})
	if err != nil {
		return fmt.Errorf("encode avatar delete request: %w", err)
	}
	req, err := c.storageRequest(ctx, http.MethodDelete, "/object/"+c.config.AvatarBucket, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create avatar delete request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrDeleteFailed, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("%w: storage responded with status %d: %s", ErrDeleteFailed, resp.StatusCode, readSnippet(resp.Body))
	}
	c.mu.Lock()
	delete(c.signedURLs, stored)
	c.mu.Unlock()
	return nil
}

// AvatarURLs signs the avatars of the private bucket in a single request. Signed URLs are cached and reused
// while they are valid for at least half of AvatarURLTTL, so clients get the same URL (and can cache the image).
func (c *S3Client) AvatarURLs(ctx context.Context, avatars []AvatarRef) ([]string, error) {
	urls := make([]string, len(avatars))
	var missing []string // object paths to sign, without duplicates
	pending := map[string][]int{}

	now := c.now()
	c.mu.Lock()
	for i, avatar := range avatars {
		switch {
		case avatar.Stored == "":
		case IsExternalPicture(avatar.Stored):
			urls[i] = avatar.Stored
		case isAvatarOf(avatar.Stored, avatar.UserID):
			if cached, ok := c.signedURLs[avatar.Stored]; ok && cached.expiresAt.Sub(now) >= c.config.AvatarURLTTL/2 {
				urls[i] = cached.url
				continue
			}
			if _, ok := pending[avatar.Stored]; !ok {
				missing = append(missing, avatar.Stored)
			}
			pending[avatar.Stored] = append(pending[avatar.Stored], i)
		}
		// anything else (a path of another user, an unknown value) is hidden
	}
	c.mu.Unlock()

	if len(missing) == 0 {
		return urls, nil
	}
	if !c.avatarsConfigured() {
		return urls, ErrStorageNotConfigured
	}

	signed, err := c.signAvatars(ctx, missing)
	if err != nil {
		return urls, err
	}

	expiresAt := now.Add(c.config.AvatarURLTTL)
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.signedURLs)+len(signed) > maxCachedSignedURLs {
		for path, cached := range c.signedURLs {
			if !cached.expiresAt.After(now) {
				delete(c.signedURLs, path)
			}
		}
	}
	for path, url := range signed {
		for _, i := range pending[path] {
			urls[i] = url
		}
		if len(c.signedURLs) < maxCachedSignedURLs {
			c.signedURLs[path] = signedURL{url: url, expiresAt: expiresAt}
		}
	}
	return urls, nil
}

// signAvatars calls POST /object/sign/{bucket} and returns the signed URL of each path that could be signed.
// Paths that do not exist in the bucket come back with an error and are left out.
func (c *S3Client) signAvatars(ctx context.Context, paths []string) (map[string]string, error) {
	body, err := json.Marshal(struct {
		ExpiresIn int      `json:"expiresIn"`
		Paths     []string `json:"paths"`
	}{ExpiresIn: int(c.config.AvatarURLTTL.Seconds()), Paths: paths})
	if err != nil {
		return nil, fmt.Errorf("encode sign request: %w", err)
	}
	req, err := c.storageRequest(ctx, http.MethodPost, "/object/sign/"+c.config.AvatarBucket, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create sign request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrSignFailed, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("%w: storage responded with status %d: %s", ErrSignFailed, resp.StatusCode, readSnippet(resp.Body))
	}

	var items []struct {
		Path      string  `json:"path"`
		SignedURL *string `json:"signedURL"`
		Error     *string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		return nil, fmt.Errorf("%w: decode response: %v", ErrSignFailed, err)
	}
	signed := make(map[string]string, len(items))
	for _, item := range items {
		if item.SignedURL == nil || *item.SignedURL == "" || !strings.HasPrefix(*item.SignedURL, "/") {
			continue
		}
		signed[item.Path] = c.config.SupabaseURL + "/storage/v1" + *item.SignedURL
	}
	return signed, nil
}

// storageRequest builds an authenticated request to the Supabase Storage API. The secret key goes in the apikey
// header; legacy service_role keys are JWTs and also go in Authorization (new sb_secret_ keys are not JWTs and
// the gateway rejects them there).
func (c *S3Client) storageRequest(ctx context.Context, method, path string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, c.config.SupabaseURL+"/storage/v1"+path, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("apikey", c.config.SecretKey)
	if strings.HasPrefix(c.config.SecretKey, "eyJ") {
		req.Header.Set("Authorization", "Bearer "+c.config.SecretKey)
	}
	return req, nil
}

func (c *S3Client) avatarsConfigured() bool {
	return c.config.SupabaseURL != "" && c.config.SecretKey != ""
}

// IsExternalPicture reports whether a profile picture is an external URL (e.g. the Google photo) instead of an
// avatar of the private bucket.
func IsExternalPicture(picture string) bool {
	return strings.HasPrefix(picture, "https://") || strings.HasPrefix(picture, "http://")
}

// isAvatarOf reports whether stored is an object path created by UploadAvatar for userID.
func isAvatarOf(stored, userID string) bool {
	m := avatarPathRegex.FindStringSubmatch(stored)
	return m != nil && userID != "" && m[1] == strings.ToLower(userID)
}

func avatarExtension(contentType string) (string, bool) {
	switch contentType {
	case "image/jpeg":
		return ".jpg", true
	case "image/png":
		return ".png", true
	case "image/webp":
		return ".webp", true
	default:
		return "", false
	}
}

// readSnippet returns the start of an error body for the logs. The body never contains the secret key.
func readSnippet(body io.Reader) string {
	b, _ := io.ReadAll(io.LimitReader(body, 256))
	return strings.TrimSpace(string(b))
}
