package mcp_client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"api-aprop/internal/mcp_server"
	"api-aprop/internal/models/event_model"
	"api-aprop/internal/repositories/catalogue_repository"
)

type fakeCatalogue struct{}

func (fakeCatalogue) Search(context.Context, catalogue_repository.SearchFilter) ([]event_model.Event, error) {
	return []event_model.Event{{ID: 4, Title: "Cinema a la fresca", Municipality: "Reus"}}, nil
}

func (fakeCatalogue) ListByIDs(context.Context, []int) ([]event_model.Event, error) {
	return nil, nil
}

func newSession(t *testing.T, token string) (Session, error) {
	t.Helper()
	server := mcp_server.NewServer(fakeCatalogue{}, func() time.Time { return time.Date(2026, 10, 19, 0, 0, 0, 0, time.UTC) })
	ts := httptest.NewServer(mcp_server.NewHandler(server, "secret"))
	t.Cleanup(ts.Close)
	return NewMCPClient(&http.Client{Timeout: 5 * time.Second}, ts.URL, token).Connect(context.Background())
}

func TestMCPClient_ListAndCallTools(t *testing.T) {
	session, err := newSession(t, "secret")
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer session.Close()

	tools, err := session.ListTools(context.Background())
	if err != nil || len(tools) != 2 {
		t.Fatalf("tools = %v, err = %v", tools, err)
	}

	result, err := session.CallTool(context.Background(), mcp_server.ToolSearchEvents, nil)
	if err != nil || result.IsError {
		t.Fatalf("result = %+v, err = %v", result, err)
	}
	var out mcp_server.SearchEventsOutput
	if err := json.Unmarshal(result.Structured, &out); err != nil || len(out.Events) != 1 || out.Events[0].ID != 4 {
		t.Fatalf("structured = %s", result.Structured)
	}
	if result.Text == "" {
		t.Errorf("the text content is what the model reads, it must not be empty")
	}

	result, err = session.CallTool(context.Background(), mcp_server.ToolGetEvent, json.RawMessage(`{"id":99}`))
	if err != nil || !result.IsError {
		t.Fatalf("missing event: result = %+v, err = %v", result, err)
	}
}

func TestMCPClient_WrongToken(t *testing.T) {
	if _, err := newSession(t, "wrong"); err == nil {
		t.Fatalf("expected the connection to be rejected")
	}
}
