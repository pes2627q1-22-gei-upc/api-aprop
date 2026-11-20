package translate_client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	defaultTimeout = 15 * time.Second
	maxResponse    = 4 << 20
)

// TranslatorClient calls POST {url}/translate on the configured translation server.
type TranslatorClient struct {
	url        string
	apiKey     string
	httpClient *http.Client
}

var _ Client = (*TranslatorClient)(nil)

// NewTranslatorClient creates the client. An empty url disables it: Translate returns ErrDisabled.
// apiKey is optional. A nil httpClient uses one with a timeout.
func NewTranslatorClient(url, apiKey string, httpClient *http.Client) *TranslatorClient {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: defaultTimeout}
	}
	return &TranslatorClient{url: strings.TrimRight(strings.TrimSpace(url), "/"), apiKey: apiKey, httpClient: httpClient}
}

type translateRequest struct {
	Q      []string `json:"q"`
	Source string   `json:"source"`
	Target string   `json:"target"`
	Format string   `json:"format"`
	APIKey string   `json:"api_key,omitempty"`
}

type translateResponse struct {
	TranslatedText []string `json:"translatedText"`
	Error          string   `json:"error"`
}

func (c *TranslatorClient) Translate(ctx context.Context, texts []string, source, target string) ([]string, error) {
	if c.url == "" {
		return nil, ErrDisabled
	}
	if len(texts) == 0 {
		return []string{}, nil
	}
	body, err := json.Marshal(translateRequest{Q: texts, Source: source, Target: target, Format: "text", APIKey: c.apiKey})
	if err != nil {
		return nil, fmt.Errorf("translate: encode request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url+"/translate", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("translate: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("translate: %w", err)
	}
	defer res.Body.Close()

	var out translateResponse
	decodeErr := json.NewDecoder(io.LimitReader(res.Body, maxResponse)).Decode(&out)
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("translate: translator answered %d: %s", res.StatusCode, out.Error)
	}
	if decodeErr != nil {
		return nil, fmt.Errorf("translate: decode response: %w", decodeErr)
	}
	if len(out.TranslatedText) != len(texts) {
		return nil, fmt.Errorf("translate: expected %d translations, got %d", len(texts), len(out.TranslatedText))
	}
	return out.TranslatedText, nil
}
