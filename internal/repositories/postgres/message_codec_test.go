package postgres

import (
	"bytes"
	"compress/flate"
	"errors"
	"strings"
	"testing"

	"api-aprop/internal/models/message_model"
)

func TestMessageCodec_RoundTrip(t *testing.T) {
	for name, text := range map[string]string{
		"short":     "Hola!",
		"accents":   "Ens veiem a l'entrada, porto l'entrada impresa 🎭",
		"long":      strings.Repeat("Recordeu que l'activitat comença puntual. ", 20),
		"max chars": strings.Repeat("à", message_model.MaxTextLength),
	} {
		t.Run(name, func(t *testing.T) {
			encoded, err := encodeMessage(text)
			if err != nil {
				t.Fatal(err)
			}
			got, err := decodeMessage(encoded)
			if err != nil {
				t.Fatal(err)
			}
			if got != text {
				t.Fatalf("round trip = %q, want %q", got, text)
			}
			// The database CHECK only accepts format byte + 1..4000 bytes
			if len(encoded) < 2 || len(encoded) > 4001 {
				t.Errorf("encoded size %d outside 2..4001", len(encoded))
			}
		})
	}
}

func TestMessageCodec_ShortTextStaysPlain(t *testing.T) {
	encoded, err := encodeMessage("Hola!")
	if err != nil {
		t.Fatal(err)
	}
	if encoded[0] != messageFormatPlain || string(encoded[1:]) != "Hola!" {
		t.Fatalf("encoded = %v, want the plain format", encoded)
	}
}

func TestMessageCodec_LongTextIsCompressed(t *testing.T) {
	text := strings.Repeat("Recordeu que l'activitat comença puntual. ", 20)
	encoded, err := encodeMessage(text)
	if err != nil {
		t.Fatal(err)
	}
	if encoded[0] != messageFormatDeflate || len(encoded) >= len(text) {
		t.Fatalf("format %d, %d bytes for a %d byte text: want compressed and smaller", encoded[0], len(encoded), len(text))
	}
}

// Rows that existed before migration 000019 are converted to format byte 0 + the old text.
func TestMessageCodec_MigratedRow(t *testing.T) {
	got, err := decodeMessage(append([]byte{0}, "missatge antic"...))
	if err != nil || got != "missatge antic" {
		t.Fatalf("decode = %q, %v", got, err)
	}
}

func TestMessageCodec_Corrupt(t *testing.T) {
	var bomb bytes.Buffer
	bomb.WriteByte(messageFormatDeflate)
	w, _ := flate.NewWriter(&bomb, flate.BestCompression)
	_, _ = w.Write(bytes.Repeat([]byte("a"), maxDecodedMessageBytes+1))
	_ = w.Close()

	for name, data := range map[string][]byte{
		"empty":          nil,
		"unknown format": {7, 'h', 'i'},
		"bad deflate":    {messageFormatDeflate, 0xff, 0xff, 0xff},
		"too long":       bomb.Bytes(),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := decodeMessage(data); !errors.Is(err, errCorruptMessage) {
				t.Fatalf("err = %v, want errCorruptMessage", err)
			}
		})
	}
}
