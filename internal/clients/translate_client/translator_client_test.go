package translate_client

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTranslate(t *testing.T) {
	var got translateRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/translate" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		_ = json.NewDecoder(r.Body).Decode(&got)
		_ = json.NewEncoder(w).Encode(map[string]any{"translatedText": []string{"Hola", "Mundo"}})
	}))
	defer srv.Close()

	c := NewTranslatorClient(srv.URL+"/", "secret", nil)
	out, err := c.Translate(context.Background(), []string{"Hola?", "Món"}, "ca", "es")
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 2 || out[0] != "Hola" || out[1] != "Mundo" {
		t.Errorf("translations = %v", out)
	}
	if got.Source != "ca" || got.Target != "es" || got.Format != "text" || got.APIKey != "secret" || len(got.Q) != 2 {
		t.Errorf("request body = %+v", got)
	}
}

func TestTranslateWithoutAPIKeyOmitsIt(t *testing.T) {
	var raw map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&raw)
		_ = json.NewEncoder(w).Encode(map[string]any{"translatedText": []string{"a"}})
	}))
	defer srv.Close()

	if _, err := NewTranslatorClient(srv.URL, "", nil).Translate(context.Background(), []string{"a"}, "ca", "en"); err != nil {
		t.Fatal(err)
	}
	if _, ok := raw["api_key"]; ok {
		t.Error("api_key must be omitted when empty")
	}
}

func TestTranslateErrors(t *testing.T) {
	tests := []struct {
		name    string
		handler http.HandlerFunc
	}{
		{"status error", func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":"bad language"}`))
		}},
		{"invalid json", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("not json")) }},
		{"wrong count", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(`{"translatedText":["a"]}`)) }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(tt.handler)
			defer srv.Close()
			if _, err := NewTranslatorClient(srv.URL, "", nil).Translate(context.Background(), []string{"a", "b"}, "ca", "es"); err == nil {
				t.Error("expected an error")
			}
		})
	}
}

func TestTranslateDisabled(t *testing.T) {
	_, err := NewTranslatorClient("", "", nil).Translate(context.Background(), []string{"a"}, "ca", "es")
	if !errors.Is(err, ErrDisabled) {
		t.Errorf("err = %v, want ErrDisabled", err)
	}
}
