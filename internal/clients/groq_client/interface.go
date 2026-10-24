package groq_client

import "context"

// Client creates chat completions with the Groq API.
type Client interface {
	ChatCompletion(ctx context.Context, req ChatCompletionRequest) (ChatCompletionResponse, error)
}
