package postgres

import (
	"bytes"
	"compress/flate"
	"errors"
	"fmt"
	"io"

	"api-aprop/internal/models/message_model"
)

// EventMessages.Message is stored as bytea: one format byte followed by the text (DP-040).
// Postgres only compresses rows over ~2 kB, which a chat message never reaches, so the text is compressed here.
const (
	messageFormatPlain   byte = 0 // UTF-8 text as is
	messageFormatDeflate byte = 1 // raw DEFLATE (compress/flate) of the UTF-8 text
)

// maxDecodedMessageBytes bounds the decompressed size: MaxTextLength characters of at most 4 bytes each.
const maxDecodedMessageBytes = message_model.MaxTextLength * 4

var errCorruptMessage = errors.New("corrupt event message")

// encodeMessage compresses the text when that makes it smaller; short messages are stored as plain text.
func encodeMessage(text string) ([]byte, error) {
	var compressed bytes.Buffer
	compressed.WriteByte(messageFormatDeflate)
	w, err := flate.NewWriter(&compressed, flate.BestCompression)
	if err != nil {
		return nil, fmt.Errorf("compress message: %w", err)
	}
	if _, err := io.WriteString(w, text); err != nil {
		return nil, fmt.Errorf("compress message: %w", err)
	}
	if err := w.Close(); err != nil {
		return nil, fmt.Errorf("compress message: %w", err)
	}
	if compressed.Len() < 1+len(text) {
		return compressed.Bytes(), nil
	}
	return append([]byte{messageFormatPlain}, text...), nil
}

// decodeMessage returns the text stored by encodeMessage.
func decodeMessage(data []byte) (string, error) {
	if len(data) == 0 {
		return "", fmt.Errorf("%w: empty", errCorruptMessage)
	}
	switch data[0] {
	case messageFormatPlain:
		return string(data[1:]), nil
	case messageFormatDeflate:
		r := flate.NewReader(bytes.NewReader(data[1:]))
		defer r.Close()
		text, err := io.ReadAll(io.LimitReader(r, maxDecodedMessageBytes+1))
		if err != nil {
			return "", fmt.Errorf("%w: %v", errCorruptMessage, err)
		}
		if len(text) > maxDecodedMessageBytes {
			return "", fmt.Errorf("%w: longer than %d bytes", errCorruptMessage, maxDecodedMessageBytes)
		}
		return string(text), nil
	default:
		return "", fmt.Errorf("%w: unknown format %d", errCorruptMessage, data[0])
	}
}
