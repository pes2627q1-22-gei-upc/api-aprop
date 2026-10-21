package gemini_client

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
	DefaultBaseURL = "https://generativelanguage.googleapis.com/v1beta"

	maxErrorBodyBytes = 2 << 10
)

type GeminiClient struct {
	httpClient *http.Client
	baseURL    string
	apiKey     string
	model      string
}

var _ Client = (*GeminiClient)(nil)

func NewGeminiClient(httpClient *http.Client, baseURL string, apiKey string, model string) *GeminiClient {
	return &GeminiClient{httpClient: httpClient, baseURL: baseURL, apiKey: apiKey, model: model}
}

type functionToolWire struct {
	Type string `json:"type"`
	FunctionTool
}

type generationConfig struct {
	ThinkingLevel string `json:"thinking_level,omitempty"`
}

type interactionRequestWire struct {
	Model             string             `json:"model"`
	Input             []json.RawMessage  `json:"input"`
	SystemInstruction string             `json:"system_instruction,omitempty"`
	Tools             []functionToolWire `json:"tools,omitempty"`
	// Conversations are stored in our database, Google does not need to keep them.
	Store            bool             `json:"store"`
	GenerationConfig generationConfig `json:"generation_config"`
}

func (c *GeminiClient) CreateInteraction(ctx context.Context, req InteractionRequest) (Interaction, error) {
	wire := interactionRequestWire{
		Model:             c.model,
		Input:             req.Input,
		SystemInstruction: req.SystemInstruction,
		Store:             false,
		// Low thinking keeps the reply under the 10 s response time requirement
		GenerationConfig: generationConfig{ThinkingLevel: "low"},
	}
	for _, t := range req.Tools {
		wire.Tools = append(wire.Tools, functionToolWire{Type: "function", FunctionTool: t})
	}
	body, err := json.Marshal(wire)
	if err != nil {
		return Interaction{}, fmt.Errorf("encode gemini request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/interactions", bytes.NewReader(body))
	if err != nil {
		return Interaction{}, fmt.Errorf("build gemini request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-goog-api-key", c.apiKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return Interaction{}, fmt.Errorf("call gemini api: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		errorBody, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrorBodyBytes))
		log.Printf("gemini api status=%d body=%s", resp.StatusCode, errorBody)
		return Interaction{}, fmt.Errorf("create interaction: %w: %d", ErrUnexpectedStatus, resp.StatusCode)
	}

	var interaction Interaction
	if err := json.NewDecoder(resp.Body).Decode(&interaction); err != nil {
		return Interaction{}, fmt.Errorf("decode gemini response: %w", err)
	}
	switch interaction.Status {
	case "completed", "requires_action":
		return interaction, nil
	default:
		return Interaction{}, fmt.Errorf("create interaction: %w: status %q", ErrInteractionFailed, interaction.Status)
	}
}
