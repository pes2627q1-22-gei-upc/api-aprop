package gemini_client

import "errors"

var (
	ErrUnexpectedStatus  = errors.New("unexpected status from the gemini api")
	ErrInteractionFailed = errors.New("gemini interaction did not complete")
)
