package gemini_client

import "context"

// Client creates interactions with the Gemini Interactions API.
type Client interface {
	CreateInteraction(ctx context.Context, req InteractionRequest) (Interaction, error)
}
