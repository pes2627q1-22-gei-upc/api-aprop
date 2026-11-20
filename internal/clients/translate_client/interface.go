// Package translate_client translates texts through an external translation service.
package translate_client

import (
	"context"
	"errors"
)

// ErrDisabled is returned when no translator URL is configured.
var ErrDisabled = errors.New("translation is disabled")

// Client translates texts between languages.
type Client interface {
	// Translate translates every text from source to target (ISO 639-1 codes) and returns them in the same order.
	Translate(ctx context.Context, texts []string, source, target string) ([]string, error)
}
