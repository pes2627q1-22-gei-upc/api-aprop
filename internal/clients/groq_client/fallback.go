package groq_client

import (
	"context"
	"errors"
	"log"
)

// GeminiBaseURL is the OpenAI compatible endpoint of Gemini, so GroqClient can talk to it too.
const GeminiBaseURL = "https://generativelanguage.googleapis.com/v1beta/openai"

// FallbackClient tries the clients in order and returns the first answer.
// The next client is tried after any failure except the end of the caller's context.
type FallbackClient struct {
	clients []Client
}

var _ Client = (*FallbackClient)(nil)

func NewFallbackClient(clients ...Client) *FallbackClient {
	return &FallbackClient{clients: clients}
}

func (f *FallbackClient) ChatCompletion(ctx context.Context, req ChatCompletionRequest) (ChatCompletionResponse, error) {
	if len(f.clients) == 0 {
		return ChatCompletionResponse{}, errors.New("no chat client configured")
	}
	var errs []error
	for i, c := range f.clients {
		resp, err := c.ChatCompletion(ctx, req)
		if err == nil {
			return resp, nil
		}
		if ctx.Err() != nil {
			return ChatCompletionResponse{}, err
		}
		log.Printf("chat client %d/%d failed: %v", i+1, len(f.clients), err)
		errs = append(errs, err)
	}
	return ChatCompletionResponse{}, errors.Join(errs...)
}
