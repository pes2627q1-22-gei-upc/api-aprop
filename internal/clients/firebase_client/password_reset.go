package firebase_client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// DefaultIdentityToolkitURL is the base URL of the Firebase Auth REST API.
const DefaultIdentityToolkitURL = "https://identitytoolkit.googleapis.com"

const identityToolkitTimeout = 10 * time.Second

// IdentityToolkitClient calls the Firebase Auth REST API with the Web API key of the project. The Admin
// SDK cannot be used to reset passwords: it only generates the link and does not send the email.
type IdentityToolkitClient struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

var _ PasswordResetSender = (*IdentityToolkitClient)(nil)

// NewIdentityToolkitClient creates the client. An empty baseURL uses DefaultIdentityToolkitURL and a nil
// httpClient uses one with a timeout of 10 s.
func NewIdentityToolkitClient(apiKey, baseURL string, httpClient *http.Client) *IdentityToolkitClient {
	if baseURL == "" {
		baseURL = DefaultIdentityToolkitURL
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: identityToolkitTimeout}
	}
	return &IdentityToolkitClient{apiKey: apiKey, baseURL: strings.TrimRight(baseURL, "/"), httpClient: httpClient}
}

type sendOobCodeRequest struct {
	RequestType string `json:"requestType"`
	Email       string `json:"email"`
}

type identityToolkitError struct {
	Error struct {
		Message string `json:"message"`
	} `json:"error"`
}

// SendPasswordResetEmail calls accounts:sendOobCode with the PASSWORD_RESET request type.
func (c *IdentityToolkitClient) SendPasswordResetEmail(ctx context.Context, email, locale string) error {
	body, err := json.Marshal(sendOobCodeRequest{RequestType: "PASSWORD_RESET", Email: email})
	if err != nil {
		return fmt.Errorf("encode password reset request: %w", err)
	}
	endpoint := c.baseURL + "/v1/accounts:sendOobCode?key=" + url.QueryEscape(c.apiKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create password reset request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if locale != "" {
		req.Header.Set("X-Firebase-Locale", locale)
	}

	res, err := c.httpClient.Do(req)
	if err != nil {
		// The error of Do includes the URL, and so the API key: it is not wrapped
		if ctxErr := ctx.Err(); ctxErr != nil {
			return fmt.Errorf("send password reset email: %w", ctxErr)
		}
		return fmt.Errorf("send password reset email: request failed")
	}
	defer res.Body.Close()

	if res.StatusCode == http.StatusOK {
		_, _ = io.Copy(io.Discard, res.Body)
		return nil
	}

	var apiErr identityToolkitError
	_ = json.NewDecoder(io.LimitReader(res.Body, 64<<10)).Decode(&apiErr)
	// The message can have details after the code, like "TOO_MANY_ATTEMPTS_TRY_LATER : ..."
	code, _, _ := strings.Cut(apiErr.Error.Message, " ")
	switch code {
	case "EMAIL_NOT_FOUND":
		return ErrEmailNotFound
	case "INVALID_EMAIL", "MISSING_EMAIL":
		return ErrInvalidEmail
	case "TOO_MANY_ATTEMPTS_TRY_LATER", "RESET_PASSWORD_EXCEED_LIMIT", "QUOTA_EXCEEDED":
		return ErrTooManyAttempts
	default:
		return fmt.Errorf("send password reset email: status %d: %q", res.StatusCode, code)
	}
}
