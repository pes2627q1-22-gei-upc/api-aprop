package mcp_server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"api-aprop/internal/models/event_model"
	"api-aprop/internal/repositories/catalogue_repository"
)

var fixedNow = time.Date(2026, 10, 19, 16, 30, 0, 0, time.UTC)

type fakeCatalogue struct {
	events    []event_model.Event
	err       error
	gotFilter catalogue_repository.SearchFilter
}

func (f *fakeCatalogue) Search(_ context.Context, filter catalogue_repository.SearchFilter) ([]event_model.Event, error) {
	f.gotFilter = filter
	return f.events, f.err
}

func (f *fakeCatalogue) ListByIDs(_ context.Context, ids []int) ([]event_model.Event, error) {
	var out []event_model.Event
	for _, id := range ids {
		for _, e := range f.events {
			if e.ID == id {
				out = append(out, e)
			}
		}
	}
	return out, f.err
}

func connect(t *testing.T, catalogue catalogue_repository.Repository) *mcp.ClientSession {
	t.Helper()
	server := NewServer(catalogue, func() time.Time { return fixedNow })
	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	ctx := context.Background()
	if _, err := server.Connect(ctx, serverTransport, nil); err != nil {
		t.Fatalf("server connect: %v", err)
	}
	session, err := mcp.NewClient(&mcp.Implementation{Name: "test"}, nil).Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("client connect: %v", err)
	}
	t.Cleanup(func() { session.Close() })
	return session
}

func callTool(t *testing.T, session *mcp.ClientSession, name string, args map[string]any) *mcp.CallToolResult {
	t.Helper()
	res, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		t.Fatalf("call %s: %v", name, err)
	}
	return res
}

func resultText(res *mcp.CallToolResult) string {
	var sb strings.Builder
	for _, c := range res.Content {
		if text, ok := c.(*mcp.TextContent); ok {
			sb.WriteString(text.Text)
		}
	}
	return sb.String()
}

func TestListTools(t *testing.T) {
	session := connect(t, &fakeCatalogue{})
	res, err := session.ListTools(context.Background(), &mcp.ListToolsParams{})
	if err != nil {
		t.Fatalf("list tools: %v", err)
	}
	names := map[string]bool{}
	for _, tool := range res.Tools {
		names[tool.Name] = true
		if tool.Annotations == nil || !tool.Annotations.ReadOnlyHint {
			t.Errorf("tool %s should be read only", tool.Name)
		}
	}
	if !names[ToolSearchEvents] || !names[ToolGetEvent] {
		t.Fatalf("tools = %v, want %s and %s", names, ToolSearchEvents, ToolGetEvent)
	}
}

func TestSearchEvents_DefaultsAndResult(t *testing.T) {
	catalogue := &fakeCatalogue{events: []event_model.Event{{
		ID: 7, Title: "Concert de jazz", Municipality: "Girona",
		StartDate: time.Date(2026, 10, 24, 0, 0, 0, 0, time.UTC), Description: strings.Repeat("a", 400),
	}}}
	session := connect(t, catalogue)

	res := callTool(t, session, ToolSearchEvents, map[string]any{"category": " concert ", "municipality": "Girona"})
	if res.IsError {
		t.Fatalf("unexpected tool error: %s", resultText(res))
	}

	got := catalogue.gotFilter
	if got.Category != "concert" || got.Municipality != "Girona" || got.Limit != defaultLimit || got.To != nil {
		t.Errorf("filter = %+v", got)
	}
	if got.From == nil || got.From.Format(DateLayout) != "2026-10-19" {
		t.Errorf("from = %v, want today", got.From)
	}

	var out SearchEventsOutput
	if err := json.Unmarshal([]byte(resultText(res)), &out); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if len(out.Events) != 1 || out.Events[0].ID != 7 || out.Events[0].StartDate != "2026-10-24" {
		t.Fatalf("events = %+v", out.Events)
	}
	if n := len([]rune(out.Events[0].Description)); n != maxDescriptionRune+1 {
		t.Errorf("description has %d runes, want it truncated", n)
	}
}

func TestSearchEvents_LimitIsCapped(t *testing.T) {
	catalogue := &fakeCatalogue{}
	session := connect(t, catalogue)
	callTool(t, session, ToolSearchEvents, map[string]any{"limit": 50, "from": "2026-11-01", "to": "2026-11-02"})
	if catalogue.gotFilter.Limit != maxLimit {
		t.Errorf("limit = %d, want %d", catalogue.gotFilter.Limit, maxLimit)
	}
	if catalogue.gotFilter.To == nil || catalogue.gotFilter.To.Format(DateLayout) != "2026-11-02" {
		t.Errorf("to = %v", catalogue.gotFilter.To)
	}
}

func TestSearchEvents_InvalidInputIsToolError(t *testing.T) {
	tests := map[string]map[string]any{
		"malformed date": {"from": "24/10/2026"},
		"reversed range": {"from": "2026-10-25", "to": "2026-10-24"},
	}
	for name, args := range tests {
		t.Run(name, func(t *testing.T) {
			res := callTool(t, connect(t, &fakeCatalogue{}), ToolSearchEvents, args)
			if !res.IsError {
				t.Fatalf("expected a tool error so the model can correct the call")
			}
		})
	}
}

func TestSearchEvents_CatalogueErrorIsToolError(t *testing.T) {
	res := callTool(t, connect(t, &fakeCatalogue{err: errors.New("db down")}), ToolSearchEvents, nil)
	if !res.IsError {
		t.Fatalf("expected a tool error")
	}
}

func TestGetEvent(t *testing.T) {
	session := connect(t, &fakeCatalogue{events: []event_model.Event{{ID: 3, Title: "Teatre"}}})

	res := callTool(t, session, ToolGetEvent, map[string]any{"id": 3})
	if res.IsError || !strings.Contains(resultText(res), "Teatre") {
		t.Fatalf("result = %s", resultText(res))
	}

	res = callTool(t, session, ToolGetEvent, map[string]any{"id": 99})
	if !res.IsError {
		t.Fatalf("expected a tool error for a missing event")
	}
}

func TestHandler_RequiresToken(t *testing.T) {
	server := NewServer(&fakeCatalogue{}, func() time.Time { return fixedNow })
	ts := httptest.NewServer(NewHandler(server, "secret"))
	defer ts.Close()

	req, _ := http.NewRequest(http.MethodPost, ts.URL, strings.NewReader(`{}`))
	req.Header.Set("Authorization", "Bearer wrong")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}
}

type bearerTransport struct{ token string }

func (b bearerTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	r = r.Clone(r.Context())
	r.Header.Set("Authorization", "Bearer "+b.token)
	return http.DefaultTransport.RoundTrip(r)
}

func TestHandler_StreamableHTTP(t *testing.T) {
	server := NewServer(&fakeCatalogue{events: []event_model.Event{{ID: 1, Title: "Dansa"}}}, func() time.Time { return fixedNow })
	ts := httptest.NewServer(NewHandler(server, "secret"))
	defer ts.Close()

	transport := &mcp.StreamableClientTransport{
		Endpoint:             ts.URL,
		HTTPClient:           &http.Client{Transport: bearerTransport{token: "secret"}},
		DisableStandaloneSSE: true,
	}
	session, err := mcp.NewClient(&mcp.Implementation{Name: "test"}, nil).Connect(context.Background(), transport, nil)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer session.Close()

	res := callTool(t, session, ToolSearchEvents, nil)
	if res.IsError || !strings.Contains(resultText(res), "Dansa") {
		t.Fatalf("result = %s", resultText(res))
	}
}
