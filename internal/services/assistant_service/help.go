package assistant_service

import (
	"context"
	_ "embed"
	"encoding/json"
	"sort"
	"strings"
	"sync"
)

const ToolGetAppHelp = "get_app_help"

// guide is the user guide of the app. Each "## topic" heading starts a topic the model can ask for.
//
//go:embed help_guide.md
var guide string

// topicScreens maps a guide topic to the screen the reply suggests opening (the reply action).
// TODO(DP-028): the client must map each value to a real screen; keep this list in sync with it.
var topicScreens = map[string]string{
	"login-and-account": "settings",
	"find-activities":   "discover",
	"agenda":            "agenda",
	"notifications":     "notifications",
	"profile":           "profile",
	"settings":          "settings",
	"interest-test":     "interest-test",
	"assistant":         "home",
}

// helpTopics maps a topic key to its text and lists the keys in order.
var helpTopics = sync.OnceValues(func() (map[string]string, []string) {
	topics := map[string]string{}
	var keys []string
	for _, section := range strings.Split("\n"+guide, "\n## ")[1:] {
		key, body, _ := strings.Cut(section, "\n")
		key = strings.TrimSpace(key)
		topics[key] = strings.TrimSpace(body)
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return topics, keys
})

// helpScreen returns the screen to suggest for a get_app_help call, or "" if there is none.
func helpScreen(arguments string) string {
	var in struct {
		Topic string `json:"topic"`
	}
	if json.Unmarshal([]byte(arguments), &in) != nil {
		return ""
	}
	return topicScreens[strings.TrimSpace(in.Topic)]
}

func (s *AssistantService) getAppHelp(ctx context.Context, userID string, arguments string) (string, error) {
	var in struct {
		Topic string `json:"topic"`
	}
	if err := unmarshalArgs(arguments, &in); err != nil {
		return "", err
	}
	topics, keys := helpTopics()
	topic := strings.TrimSpace(in.Topic)
	text, ok := topics[topic]
	if !ok {
		return "", toolError{"unknown topic; use one of: " + strings.Join(keys, ", ")}
	}
	out := map[string]any{"topic": topic, "guide": text}
	// The user's own state lets the model skip steps already done; if it cannot be read, the guide alone is enough.
	if topic == "interest-test" && s.preferences != nil {
		if done, err := s.preferences.HasCompletedTest(ctx, userID); err == nil {
			out["user-has-done-test"] = done
		}
	}
	b, _ := json.Marshal(out)
	return string(b), nil
}
