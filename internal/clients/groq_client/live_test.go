//go:build eval

package groq_client

import (
	"context"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

// TestLiveGemini checks the Gemini backup with one real call:
//
//	GEMINI_API_KEY=... go test -tags eval -run TestLiveGemini -v ./internal/clients/groq_client/
func TestLiveGemini(t *testing.T) {
	key := os.Getenv("GEMINI_API_KEY")
	if key == "" {
		t.Skip("set GEMINI_API_KEY")
	}
	model := os.Getenv("GEMINI_MODEL")
	if model == "" {
		model = "gemini-3.8-flash"
	}
	c := NewGroqClient(&http.Client{Timeout: 30 * time.Second}, GeminiBaseURL, key, model)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	resp, err := c.ChatCompletion(ctx, ChatCompletionRequest{Messages: []Message{UserMessage("Reply with the single word: ok")}})
	if err != nil {
		t.Fatalf("gemini call failed: %v", err)
	}
	if len(resp.Choices) == 0 || !strings.Contains(strings.ToLower(resp.Choices[0].Message.Content), "ok") {
		t.Fatalf("unexpected response: %+v", resp)
	}
}
