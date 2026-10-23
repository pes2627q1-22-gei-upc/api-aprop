package groq_client

import "errors"

var (
	ErrUnexpectedStatus = errors.New("groq: unexpected status code")
	ErrEmptyChoices     = errors.New("groq: no choices in response")
)
