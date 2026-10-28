package groq_client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

const (
	DefaultBaseURL = "https://api.groq.com/openai/v1"
	DefaultModel   = "openai/gpt-oss-120b"

	maxErrorBodyBytes = 2 << 10
)

type GroqClient struct {
	httpClient *http.Client
	baseURL    string
	apiKey     string
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
	return &GroqClient{
		httpClient: httpClient,
		baseURL:    baseURL,
		apiKey:     apiKey,
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

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return ChatCompletionResponse{}, fmt.Errorf("build groq request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	httpReq.Header.Set("User-Agent", "api-aprop/1.0")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return ChatCompletionResponse{}, fmt.Errorf("call groq api: %w", err)
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
