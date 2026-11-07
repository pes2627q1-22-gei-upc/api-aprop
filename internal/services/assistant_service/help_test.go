package assistant_service

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestHelpTopics(t *testing.T) {
	topics, keys := helpTopics()
	if len(keys) < 10 || len(keys) != len(topics) {
		t.Fatalf("keys = %v", keys)
	}
	for _, k := range keys {
		if strings.TrimSpace(topics[k]) == "" {
			t.Errorf("topic %q is empty", k)
		}
	}
	if !strings.Contains(topics["sign-up"], "M'hi apunto") {
		t.Errorf("sign-up = %q", topics["sign-up"])
	}
}

func TestCallTool_GetAppHelp(t *testing.T) {
	s := newService(nil, nil, &fakeCatalogue{})
	text, search, err := s.callTool(context.Background(), validUserID, time.Now(), ToolGetAppHelp, `{"topic":"sign-up"}`)
	if err != nil || search != nil || !strings.Contains(text, "M'hi apunto") {
		t.Fatalf("text=%q search=%v err=%v", text, search, err)
	}
	text, _, err = s.callTool(context.Background(), validUserID, time.Now(), ToolGetAppHelp, `{"topic":"nope"}`)
	if err != nil || !strings.Contains(text, `"is_error":true`) || !strings.Contains(text, "sign-up") {
		t.Fatalf("unknown topics must give the model the valid ones: %q, %v", text, err)
	}
}

func TestToolDefinitions_HelpTopicsAreAnEnum(t *testing.T) {
	for _, tool := range toolDefinitions() {
		if tool.Function.Name != ToolGetAppHelp {
			continue
		}
		enum := tool.Function.Parameters.(map[string]any)["properties"].(map[string]any)["topic"].(map[string]any)["enum"].([]string)
		if len(enum) == 0 {
			t.Fatal("empty enum")
		}
		return
	}
	t.Fatal("get_app_help is not offered")
}
