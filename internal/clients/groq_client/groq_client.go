package groq_client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync/atomic"
)

const (
	DefaultBaseURL = "https://api.groq.com/openai/v1"
	DefaultModel   = "openai/gpt-oss-120b"

	maxErrorBodyBytes = 2 << 10
)

type GroqClient struct {
	httpClient *http.Client
	baseURL    string
	apiKeys    []string
	keyIndex   atomic.Uint64
	model      string
}

var _ Client = (*GroqClient)(nil)

func NewGroqClient(httpClient *http.Client, baseURL string, apiKey string, model string) *GroqClient {
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	if model == "" {
		model = DefaultModel
	}
	rawKeys := strings.Split(apiKey, ",")
	cleanKeys := make([]string, 0, len(rawKeys))
	for _, k := range rawKeys {
		k = strings.TrimSpace(k)
		if k != "" {
			cleanKeys = append(cleanKeys, k)
		}
	}
	return &GroqClient{
		httpClient: httpClient,
		baseURL:    baseURL,
		apiKeys:    cleanKeys,
		model:      model,
	}
}

func (c *GroqClient) ChatCompletion(ctx context.Context, req ChatCompletionRequest) (ChatCompletionResponse, error) {
	if req.Model == "" {
		req.Model = c.model
	}

	body, err := json.Marshal(req)
	if err != nil {
		return ChatCompletionResponse{}, fmt.Errorf("encode groq request: %w", err)
	}

	numKeys := len(c.apiKeys)
	if numKeys == 0 {
		return ChatCompletionResponse{}, errors.New("no groq api key configured")
	}

	startIdx := int(c.keyIndex.Add(1)-1) % numKeys

	for attempt := 0; attempt < numKeys; attempt++ {
		idx := (startIdx + attempt) % numKeys
		key := c.apiKeys[idx]

		httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(body))
		if err != nil {
			return ChatCompletionResponse{}, fmt.Errorf("build groq request: %w", err)
		}
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("Authorization", "Bearer "+key)
		httpReq.Header.Set("User-Agent", "api-aprop/1.0")

		resp, err := c.httpClient.Do(httpReq)
		if err != nil {
			return ChatCompletionResponse{}, fmt.Errorf("call groq api: %w", err)
		}

		if resp.StatusCode == http.StatusTooManyRequests && numKeys > 1 && attempt < numKeys-1 {
			resp.Body.Close()
			log.Printf("groq api key %d/%d rate limited (429), rotating to next key...", idx+1, numKeys)
			continue
		}

		defer resp.Body.Close()

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			errorBody, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrorBodyBytes))
			log.Printf("groq api status=%d body=%s", resp.StatusCode, errorBody)
			return ChatCompletionResponse{}, fmt.Errorf("%w: %d", ErrUnexpectedStatus, resp.StatusCode)
		}

		var completion ChatCompletionResponse
		if err := json.NewDecoder(resp.Body).Decode(&completion); err != nil {
			return ChatCompletionResponse{}, fmt.Errorf("decode groq response: %w", err)
		}

		if len(completion.Choices) == 0 {
			return ChatCompletionResponse{}, ErrEmptyChoices
		}

		return completion, nil
	}

	return ChatCompletionResponse{}, fmt.Errorf("%w: 429 all keys rate limited", ErrUnexpectedStatus)
}
