package gemini_client

import (
	"encoding/json"
	"strings"
)

// InteractionRequest is a stateless interaction: the whole history is sent in Input.
type InteractionRequest struct {
	SystemInstruction string
	// Input holds the conversation steps. Model steps must be sent back exactly as received
	// because they carry the thought signatures.
	Input []json.RawMessage
	Tools []FunctionTool
}

// FunctionTool declares a function the model may call. Parameters is a JSON schema object.
type FunctionTool struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Parameters  any    `json:"parameters,omitempty"`
}

// Interaction is the response of the API.
type Interaction struct {
	ID     string            `json:"id"`
	Status string            `json:"status"`
	Steps  []json.RawMessage `json:"steps"`
}

// FunctionCall is a request of the model to call a tool.
type FunctionCall struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

type stepHeader struct {
	Type string `json:"type"`
}

type textContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type contentStep struct {
	Type    string        `json:"type"`
	Content []textContent `json:"content"`
}

type functionResultStep struct {
	Type    string        `json:"type"`
	Name    string        `json:"name"`
	CallID  string        `json:"call_id"`
	Result  []textContent `json:"result"`
	IsError bool          `json:"is_error,omitempty"`
}

// UserInputStep is a text message of the user.
func UserInputStep(text string) json.RawMessage {
	return mustMarshal(contentStep{Type: "user_input", Content: []textContent{{Type: "text", Text: text}}})
}

// ModelOutputStep is a previous text reply of the model, used to replay the conversation history.
func ModelOutputStep(text string) json.RawMessage {
	return mustMarshal(contentStep{Type: "model_output", Content: []textContent{{Type: "text", Text: text}}})
}

// FunctionResultStep is the result of a function call requested by the model.
func FunctionResultStep(call FunctionCall, result string, isError bool) json.RawMessage {
	return mustMarshal(functionResultStep{
		Type:    "function_result",
		Name:    call.Name,
		CallID:  call.ID,
		Result:  []textContent{{Type: "text", Text: result}},
		IsError: isError,
	})
}

// FunctionCalls returns the function calls requested in the interaction.
func (i Interaction) FunctionCalls() []FunctionCall {
	var calls []FunctionCall
	for _, raw := range i.Steps {
		if stepType(raw) != "function_call" {
			continue
		}
		var call FunctionCall
		if err := json.Unmarshal(raw, &call); err == nil {
			calls = append(calls, call)
		}
	}
	return calls
}

// OutputText joins the text of the model output steps.
func (i Interaction) OutputText() string {
	var parts []string
	for _, raw := range i.Steps {
		if stepType(raw) != "model_output" {
			continue
		}
		var step contentStep
		if err := json.Unmarshal(raw, &step); err != nil {
			continue
		}
		for _, c := range step.Content {
			if c.Type == "text" && c.Text != "" {
				parts = append(parts, c.Text)
			}
		}
	}
	return strings.TrimSpace(strings.Join(parts, "\n"))
}

func stepType(raw json.RawMessage) string {
	var h stepHeader
	_ = json.Unmarshal(raw, &h)
	return h.Type
}

// mustMarshal only receives values of the types above, which always marshal.
func mustMarshal(v any) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}
