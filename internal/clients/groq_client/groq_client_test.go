package groq_client

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newTestServer(t *testing.T, status int, response string, inspect func(*http.Request, map[string]any)) *GroqClient {
	t.Helper()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var decoded map[string]any
		if err := json.Unmarshal(body, &decoded); err != nil {
			t.Errorf("request body is not JSON: %v", err)
		}
		if inspect != nil {
			inspect(r, decoded)
		}
		w.WriteHeader(status)
		io.WriteString(w, response)
	}))
	t.Cleanup(ts.Close)
	return NewGroqClient(ts.Client(), ts.URL, "test-groq-key", "groq-test-model")
}

func TestChatCompletion_Request(t *testing.T) {
	client := newTestServer(t, http.StatusOK, `{"id":"1","choices":[{"index":0,"message":{"role":"assistant","content":"Hola"},"finish_reason":"stop"}]}`, func(r *http.Request, body map[string]any) {
		if r.Method != http.MethodPost || r.URL.Path != "/chat/completions" {
			t.Errorf("request = %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-groq-key" {
			t.Errorf("auth header = %q", got)
		}
		if body["model"] != "groq-test-model" {
			t.Errorf("model = %v", body["model"])
		}
		tools := body["tools"].([]any)
		tool := tools[0].(map[string]any)
		if tool["type"] != "function" {
			t.Errorf("tool type = %v", tool["type"])
		}
		fn := tool["function"].(map[string]any)
		if fn["name"] != "search_events" {
			t.Errorf("fn name = %v", fn["name"])
		}
		messages := body["messages"].([]any)
		if len(messages) != 2 {
			t.Errorf("messages count = %d", len(messages))
		}
	})

	resp, err := client.ChatCompletion(context.Background(), ChatCompletionRequest{
		Messages: []Message{
			SystemMessage("be nice"),
			UserMessage("hola"),
		},
		Tools: []Tool{
			{
				Type: "function",
				Function: ToolDefinition{
					Name:        "search_events",
					Description: "Search events",
					Parameters:  map[string]any{"type": "object"},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Choices) != 1 || resp.Choices[0].Message.Content != "Hola" {
		t.Errorf("resp = %+v", resp)
	}
}

func TestChatCompletion_ToolCalls(t *testing.T) {
	client := newTestServer(t, http.StatusOK, `{
		"id": "1",
		"choices": [{
			"index": 0,
			"message": {
				"role": "assistant",
				"tool_calls": [{
					"id": "call_1",
					"type": "function",
					"function": {
						"name": "search_events",
						"arguments": "{\"municipality\":\"Girona\"}"
					}
				}]
			},
			"finish_reason": "tool_calls"
		}]
	}`, nil)

	resp, err := client.ChatCompletion(context.Background(), ChatCompletionRequest{
		Messages: []Message{UserMessage("hola")},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Choices) != 1 {
		t.Fatalf("choices = %+v", resp.Choices)
	}
	msg := resp.Choices[0].Message
	if len(msg.ToolCalls) != 1 {
		t.Fatalf("tool calls = %+v", msg.ToolCalls)
	}
	tc := msg.ToolCalls[0]
	if tc.ID != "call_1" || tc.Function.Name != "search_events" || tc.Function.Arguments != `{"municipality":"Girona"}` {
		t.Errorf("tool call = %+v", tc)
	}
}

func TestChatCompletion_Errors(t *testing.T) {
	tests := map[string]struct {
		status   int
		response string
		want     error
	}{
		"http error":    {http.StatusTooManyRequests, `{"error":"quota"}`, ErrUnexpectedStatus},
		"empty choices": {http.StatusOK, `{"id":"1","choices":[]}`, ErrEmptyChoices},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			client := newTestServer(t, tt.status, tt.response, nil)
			_, err := client.ChatCompletion(context.Background(), ChatCompletionRequest{
				Messages: []Message{UserMessage("hola")},
			})
			if !errors.Is(err, tt.want) {
				t.Fatalf("error = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestSanitizeSchema(t *testing.T) {
	schema := map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"$schema":              "https://json-schema.org/draft/2020-12/schema",
		"properties": map[string]any{
			"tags": map[string]any{"type": "array", "items": map[string]any{"type": "string", "pattern": "x"}},
		},
		"required": []string{"tags"},
	}
	got, _ := json.Marshal(SanitizeSchema(schema))
	want := `{"properties":{"tags":{"items":{"type":"string"},"type":"array"}},"required":["tags"],"type":"object"}`
	if string(got) != want {
		t.Fatalf("schema = %s\nwant      %s", got, want)
	}
}
