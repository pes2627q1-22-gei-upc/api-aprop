package gemini_client

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newTestServer(t *testing.T, status int, response string, inspect func(*http.Request, map[string]any)) *GeminiClient {
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
	return NewGeminiClient(ts.Client(), ts.URL, "test-key", "gemini-test")
}

func TestCreateInteraction_Request(t *testing.T) {
	client := newTestServer(t, http.StatusOK, `{"id":"1","status":"completed","steps":[]}`, func(r *http.Request, body map[string]any) {
		if r.Method != http.MethodPost || r.URL.Path != "/interactions" {
			t.Errorf("request = %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("x-goog-api-key"); got != "test-key" {
			t.Errorf("api key header = %q", got)
		}
		if body["model"] != "gemini-test" || body["store"] != false || body["system_instruction"] != "be nice" {
			t.Errorf("body = %v", body)
		}
		tools := body["tools"].([]any)
		tool := tools[0].(map[string]any)
		if tool["type"] != "function" || tool["name"] != "search_events" {
			t.Errorf("tool = %v", tool)
		}
		input := body["input"].([]any)
		if len(input) != 2 || input[0].(map[string]any)["type"] != "user_input" || input[1].(map[string]any)["type"] != "thought" {
			t.Errorf("input = %v", input)
		}
	})

	_, err := client.CreateInteraction(context.Background(), InteractionRequest{
		SystemInstruction: "be nice",
		Input:             []json.RawMessage{UserInputStep("hola"), json.RawMessage(`{"type":"thought","signature":"abc"}`)},
		Tools:             []FunctionTool{{Name: "search_events", Parameters: map[string]any{"type": "object"}}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCreateInteraction_FunctionCallsAndText(t *testing.T) {
	client := newTestServer(t, http.StatusOK, `{"id":"1","status":"requires_action","steps":[
		{"type":"thought","signature":"sig"},
		{"type":"function_call","id":"call_1","name":"search_events","arguments":{"municipality":"Girona"}},
		{"type":"model_output","content":[{"type":"text","text":"Busco "},{"type":"text","text":"concerts"}]}
	]}`, nil)

	interaction, err := client.CreateInteraction(context.Background(), InteractionRequest{Input: []json.RawMessage{UserInputStep("hola")}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	calls := interaction.FunctionCalls()
	if len(calls) != 1 || calls[0].ID != "call_1" || calls[0].Name != "search_events" || string(calls[0].Arguments) != `{"municipality":"Girona"}` {
		t.Fatalf("calls = %+v", calls)
	}
	if got := interaction.OutputText(); got != "Busco \nconcerts" {
		t.Errorf("output text = %q", got)
	}
	if len(interaction.Steps) != 3 || string(interaction.Steps[0]) != `{"type":"thought","signature":"sig"}` {
		t.Errorf("steps must be kept as received: %s", interaction.Steps[0])
	}
}

func TestCreateInteraction_Errors(t *testing.T) {
	tests := map[string]struct {
		status   int
		response string
		want     error
	}{
		"http error":         {http.StatusTooManyRequests, `{"error":"quota"}`, ErrUnexpectedStatus},
		"failed interaction": {http.StatusOK, `{"id":"1","status":"failed","steps":[]}`, ErrInteractionFailed},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			client := newTestServer(t, tt.status, tt.response, nil)
			_, err := client.CreateInteraction(context.Background(), InteractionRequest{Input: []json.RawMessage{UserInputStep("hola")}})
			if !errors.Is(err, tt.want) {
				t.Fatalf("error = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestFunctionResultStep(t *testing.T) {
	step := FunctionResultStep(FunctionCall{ID: "call_1", Name: "search_events"}, `{"events":[]}`, true)
	var decoded map[string]any
	if err := json.Unmarshal(step, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded["type"] != "function_result" || decoded["call_id"] != "call_1" || decoded["name"] != "search_events" || decoded["is_error"] != true {
		t.Fatalf("step = %s", step)
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
